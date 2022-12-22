package index_test

import (
	"errors"
	"github.com/mrdunski/accumulation-zone/index"
	"github.com/mrdunski/accumulation-zone/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"io"
	"os"
	"strings"
	"testing"
)

type matchingFile struct {
	file model.IdentifiableHashedFile
}

func (m matchingFile) Match(actual interface{}) (success bool, err error) {
	actualFile, isFile := actual.(model.IdentifiableHashedFile)
	if !isFile {
		return false, errors.New("expected file")
	}

	return m.file.Path() == actualFile.Path() &&
		m.file.Hash() == actualFile.Hash() &&
		m.file.ChangeId() == actualFile.ChangeId(), nil
}

func (m matchingFile) FailureMessage(_ interface{}) (message string) {
	return ""
}

func (m matchingFile) NegatedFailureMessage(_ interface{}) (message string) {
	return ""
}

type entryWithContent struct {
	index.Entry
}

func (e entryWithContent) Content() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func newEntry(path, hash, changeId string) entryWithContent {
	return entryWithContent{Entry: index.NewEntry(path, hash, changeId)}
}

func TestIndex(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "index")
}

var _ = Describe("Index", func() {
	Describe("CalculateChanges", func() {
		Context("no duplicated entries in index", func() {
			var entries []index.Entry

			BeforeEach(func() {
				entries = []index.Entry{
					index.NewEntry("test1", "h1", ""),
					index.NewEntry("test2", "h2", ""),
					index.NewEntry("test3", "h3", ""),
				}
			})
			When("index and files are the same", func() {
				var files []model.FileWithContent

				BeforeEach(func() {
					files = make([]model.FileWithContent, 0, len(entries))
					for _, entry := range entries {
						files = append(files, entryWithContent{Entry: entry})
					}
				})

				It("finds no changes", func() {
					i := index.New(entries)
					changes := i.CalculateChanges(files)
					Expect(changes.Additions).To(BeEmpty())
					Expect(changes.Deletions).To(BeEmpty())
				})
			})

			When("there are no files", func() {
				var files []model.FileWithContent

				BeforeEach(func() {
					files = make([]model.FileWithContent, 0)
				})

				It("All entries marked as deleted", func() {
					i := index.New(entries)
					changes := i.CalculateChanges(files)
					Expect(changes.Deletions).To(ConsistOf(
						model.FileDeleted{IdentifiableHashedFile: entries[0]},
						model.FileDeleted{IdentifiableHashedFile: entries[1]},
						model.FileDeleted{IdentifiableHashedFile: entries[2]},
					))
				})
			})

			When("file has been modified", func() {
				var files []model.FileWithContent

				BeforeEach(func() {
					files = []model.FileWithContent{
						newEntry("test1", "h1-new", ""),
						newEntry("test2", "h2", ""),
						newEntry("test3", "h3", ""),
					}
				})

				It("finds deleted entry from index and added file", func() {
					i := index.New(entries)
					changes := i.CalculateChanges(files)
					Expect(changes.Deletions).To(ConsistOf(
						model.FileDeleted{IdentifiableHashedFile: entries[0]},
					))
					Expect(changes.Additions).To(ConsistOf(
						model.FileAdded{FileWithContent: files[0]},
					))
				})
			})

			When("file has been added", func() {
				var files []model.FileWithContent

				BeforeEach(func() {
					files = []model.FileWithContent{
						newEntry("test1", "h1", ""),
						newEntry("test2", "h2", ""),
						newEntry("test3", "h3", ""),
						newEntry("test4", "h4", ""),
					}
				})

				It("finds deleted entry from index and added file", func() {
					i := index.New(entries)
					changes := i.CalculateChanges(files)
					Expect(changes.Deletions).To(BeEmpty())
					Expect(changes.Additions).To(ConsistOf(
						model.FileAdded{FileWithContent: files[3]},
					))
				})
			})
		})

		Context("duplicated entries in index", func() {
			var entries []index.Entry

			BeforeEach(func() {
				entries = []index.Entry{
					index.NewEntry("test1", "h1a", ""),
					index.NewEntry("test1", "h1b", ""),
					index.NewEntry("test2", "h2", ""),
					index.NewEntry("test3", "h3", ""),
				}
			})

			When("file has been modified", func() {
				var files []model.FileWithContent

				BeforeEach(func() {
					files = []model.FileWithContent{
						newEntry("test1", "h1-new", ""),
						newEntry("test2", "h2", ""),
						newEntry("test3", "h3", ""),
					}
				})

				It("finds deleted entries from index and added file", func() {
					i := index.New(entries)
					changes := i.CalculateChanges(files)
					Expect(changes.Deletions).To(ConsistOf(
						model.FileDeleted{IdentifiableHashedFile: entries[0]},
						model.FileDeleted{IdentifiableHashedFile: entries[1]},
					))
					Expect(changes.Additions).To(ConsistOf(
						model.FileAdded{FileWithContent: files[0]},
					))
				})
			})

			When("file hash is in the index", func() {
				var files []model.FileWithContent

				BeforeEach(func() {
					files = []model.FileWithContent{
						newEntry("test1", "h1b", ""),
						newEntry("test2", "h2", ""),
						newEntry("test3", "h3", ""),
					}
				})

				It("keeps matching entry from index", func() {
					i := index.New(entries)
					changes := i.CalculateChanges(files)
					Expect(changes.Deletions).To(ConsistOf(
						model.FileDeleted{IdentifiableHashedFile: entries[0]},
					))
					Expect(changes.Additions).To(BeEmpty())
				})
			})
		})
	})

	Describe("CommitChange", func() {
		Context("with test entry in index", func() {
			var i index.Index
			var testEntry entryWithContent

			BeforeEach(func() {
				testEntry = newEntry("test1", "h1", "123")
				i = index.New([]index.Entry{testEntry.Entry})
			})

			It("should commit delete", func() {
				deletion := index.NewEntry("test1", "h1", "123")
				err := i.CommitDelete("123", deletion)
				Expect(err).NotTo(HaveOccurred())

				changesAfterCommit := i.CalculateChanges([]model.FileWithContent{})
				Expect(changesAfterCommit.Deletions).To(BeEmpty())
				Expect(changesAfterCommit.Additions).To(BeEmpty())
			})

			It("should not commit delete with wrong change id", func() {
				deletion := index.NewEntry("test1", "h1", "234")
				err := i.CommitDelete("234", deletion)
				Expect(err).To(HaveOccurred())

				changesAfterCommit := i.CalculateChanges([]model.FileWithContent{})
				Expect(changesAfterCommit.Deletions).To(ConsistOf(matchingFile{file: testEntry}))
				Expect(changesAfterCommit.Additions).To(BeEmpty())
			})

			It("should commit add", func() {
				entryToAdd := newEntry("test2", "h2", "")
				err := i.CommitAdd("234", entryToAdd)
				Expect(err).NotTo(HaveOccurred())

				changesAfterCommit := i.CalculateChanges([]model.FileWithContent{entryToAdd, testEntry})
				Expect(changesAfterCommit.Additions).To(BeEmpty())
				Expect(changesAfterCommit.Deletions).To(BeEmpty())
			})

			It("should not commit add for duplicated entry", func() {
				entryToAdd := newEntry("test1", "h2", "123")
				err := i.CommitAdd("123", entryToAdd)
				Expect(err).To(HaveOccurred())

				changesAfterCommit := i.CalculateChanges([]model.FileWithContent{entryToAdd, testEntry})
				Expect(changesAfterCommit.Additions).To(ConsistOf(model.FileAdded{FileWithContent: entryToAdd}))
				Expect(changesAfterCommit.Deletions).To(BeEmpty())
			})
		})
	})

	Describe("File access", func() {
		var i index.Index
		var temp *os.File

		BeforeEach(func() {
			var err error
			temp, err = os.CreateTemp("", "*-testindex.log")
			Expect(err).NotTo(HaveOccurred())
			i, err = index.LoadIndexFile(temp.Name())
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err := temp.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should add and remove file", func() {
			addition := newEntry("test1", "h1", "123")
			deletion := newEntry("test1", "h1", "123")

			err := i.CommitAdd("123", addition)
			Expect(err).NotTo(HaveOccurred())
			changesAfterAddition := i.CalculateChanges([]model.FileWithContent{addition})
			Expect(changesAfterAddition.Additions).To(BeEmpty())
			Expect(changesAfterAddition.Deletions).To(BeEmpty())

			err = i.CommitDelete("123", deletion)
			Expect(err).NotTo(HaveOccurred())
			changesAfterDeletion := i.CalculateChanges([]model.FileWithContent{})
			Expect(changesAfterDeletion.Deletions).To(BeEmpty())
			Expect(changesAfterDeletion.Additions).To(BeEmpty())
		})

		It("should reload changes", func() {
			addition := newEntry("test1", "h1", "123")

			err := i.CommitAdd("123", addition)
			Expect(err).NotTo(HaveOccurred())

			i, err = index.LoadIndexFile(temp.Name())
			Expect(err).NotTo(HaveOccurred())
			changesAfterAddition := i.CalculateChanges([]model.FileWithContent{addition})
			Expect(changesAfterAddition.Additions).To(BeEmpty())
			Expect(changesAfterAddition.Deletions).To(BeEmpty())
		})
	})
})
