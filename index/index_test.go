package index_test

import (
	"github.com/mrdunski/accumulation-zone/index"
	"github.com/mrdunski/accumulation-zone/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"os"
	"testing"
)

type changeMatcher struct {
	model.Change
}

func (c changeMatcher) Match(actual interface{}) (success bool, err error) {
	change, ok := actual.(model.Change)
	return ok && c.ChangeType == change.ChangeType && c.Hash() == change.Hash() && c.Path() == change.Path(), nil
}

func (c changeMatcher) FailureMessage(_ interface{}) string {
	return ""
}

func (c changeMatcher) NegatedFailureMessage(_ interface{}) string {
	return ""
}

func matchingChange(change model.Change) types.GomegaMatcher {
	return changeMatcher{Change: change}
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
				var files []model.HashedFile

				BeforeEach(func() {
					files = make([]model.HashedFile, 0, len(entries))
					for _, entry := range entries {
						files = append(files, entry)
					}
				})

				It("finds no changes", func() {
					i := index.New(entries)
					changes := i.CalculateChanges(files)
					Expect(changes).To(BeEmpty())
				})
			})

			When("there are no files", func() {
				var files []model.HashedFile

				BeforeEach(func() {
					files = make([]model.HashedFile, 0)
				})

				It("All entries marked as deleted", func() {
					i := index.New(entries)
					changes := i.CalculateChanges(files)
					Expect(changes).To(HaveLen(len(entries)))
					Expect(changes).To(ContainElements(
						model.Change{ChangeType: model.Deleted, HashedFile: entries[0]},
						model.Change{ChangeType: model.Deleted, HashedFile: entries[1]},
						model.Change{ChangeType: model.Deleted, HashedFile: entries[2]},
					))
				})
			})

			When("file has been modified", func() {
				var files []model.HashedFile

				BeforeEach(func() {
					files = []model.HashedFile{
						index.NewEntry("test1", "h1-new", ""),
						index.NewEntry("test2", "h2", ""),
						index.NewEntry("test3", "h3", ""),
					}
				})

				It("finds deleted entry from index and added file", func() {
					i := index.New(entries)
					changes := i.CalculateChanges(files)
					Expect(changes).To(HaveLen(2))
					Expect(changes).To(ContainElements(
						model.Change{ChangeType: model.Deleted, HashedFile: entries[0]},
						model.Change{ChangeType: model.Added, HashedFile: files[0]},
					))
				})
			})

			When("file has been added", func() {
				var files []model.HashedFile

				BeforeEach(func() {
					files = []model.HashedFile{
						index.NewEntry("test1", "h1", ""),
						index.NewEntry("test2", "h2", ""),
						index.NewEntry("test3", "h3", ""),
						index.NewEntry("test4", "h4", ""),
					}
				})

				It("finds deleted entry from index and added file", func() {
					i := index.New(entries)
					changes := i.CalculateChanges(files)
					Expect(changes).To(HaveLen(1))
					Expect(changes).To(ContainElements(
						model.Change{ChangeType: model.Added, HashedFile: files[3]},
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
				var files []model.HashedFile

				BeforeEach(func() {
					files = []model.HashedFile{
						index.NewEntry("test1", "h1-new", ""),
						index.NewEntry("test2", "h2", ""),
						index.NewEntry("test3", "h3", ""),
					}
				})

				It("finds deleted entries from index and added file", func() {
					i := index.New(entries)
					changes := i.CalculateChanges(files)
					Expect(changes).To(ContainElements(
						model.Change{ChangeType: model.Deleted, HashedFile: entries[0]},
						model.Change{ChangeType: model.Deleted, HashedFile: entries[1]},
						model.Change{ChangeType: model.Added, HashedFile: files[0]},
					))
					Expect(changes).To(HaveLen(3))
				})
			})

			When("file hash is in the index", func() {
				var files []model.HashedFile

				BeforeEach(func() {
					files = []model.HashedFile{
						index.NewEntry("test1", "h1b", ""),
						index.NewEntry("test2", "h2", ""),
						index.NewEntry("test3", "h3", ""),
					}
				})

				It("keeps matching entry from index", func() {
					i := index.New(entries)
					changes := i.CalculateChanges(files)
					Expect(changes).To(ContainElement(
						model.Change{ChangeType: model.Deleted, HashedFile: entries[0]},
					))
					Expect(changes).To(HaveLen(1))
				})
			})
		})
	})

	Describe("CommitChange", func() {
		var i index.Index
		var testEntry index.Entry

		BeforeEach(func() {
			testEntry = index.NewEntry("test1", "h1", "123")
			i = index.New([]index.Entry{testEntry})
		})

		It("should commit delete", func() {
			deletion := model.Change{ChangeType: model.Deleted, HashedFile: index.NewEntry("test1", "h1", "123")}
			err := i.CommitChange("123", deletion)
			Expect(err).NotTo(HaveOccurred())

			changesAfterCommit := i.CalculateChanges([]model.HashedFile{})
			Expect(changesAfterCommit).To(BeEmpty())
		})

		It("should not commit delete with wrong change id", func() {
			deletion := model.Change{ChangeType: model.Deleted, HashedFile: index.NewEntry("test1", "h1", "234")}
			err := i.CommitChange("234", deletion)
			Expect(err).To(HaveOccurred())

			changesAfterCommit := i.CalculateChanges([]model.HashedFile{})
			Expect(changesAfterCommit).
				To(ContainElement(matchingChange(model.Change{ChangeType: model.Deleted, HashedFile: testEntry})))
		})

		It("should commit add", func() {
			entryToAdd := index.NewEntry("test2", "h2", "")
			addition := model.Change{ChangeType: model.Added, HashedFile: entryToAdd}
			err := i.CommitChange("234", addition)
			Expect(err).NotTo(HaveOccurred())

			changesAfterCommit := i.CalculateChanges([]model.HashedFile{entryToAdd, testEntry})
			Expect(changesAfterCommit).To(BeEmpty())
		})

		It("should not commit add for duplicated entry", func() {
			entryToAdd := index.NewEntry("test1", "h2", "123")
			addition := model.Change{ChangeType: model.Added, HashedFile: entryToAdd}
			err := i.CommitChange("123", addition)
			Expect(err).To(HaveOccurred())

			changesAfterCommit := i.CalculateChanges([]model.HashedFile{entryToAdd, testEntry})
			Expect(changesAfterCommit).
				To(ContainElement(matchingChange(model.Change{ChangeType: model.Added, HashedFile: entryToAdd})))
		})

		It("fails on unsupported change", func() {
			weirdChange := model.Change{ChangeType: "weird"}
			err := i.CommitChange("123", weirdChange)
			Expect(err).To(HaveOccurred())
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
			addition := model.Change{ChangeType: model.Added, HashedFile: index.NewEntry("test1", "h1", "123")}
			deletion := model.Change{ChangeType: model.Deleted, HashedFile: index.NewEntry("test1", "h1", "123")}

			err := i.CommitChange("123", addition)
			Expect(err).NotTo(HaveOccurred())
			changesAfterAddition := i.CalculateChanges([]model.HashedFile{addition})
			Expect(changesAfterAddition).To(BeEmpty())

			err = i.CommitChange("123", deletion)
			Expect(err).NotTo(HaveOccurred())
			changesAfterDeletion := i.CalculateChanges([]model.HashedFile{})
			Expect(changesAfterDeletion).To(BeEmpty())
		})

		It("should reload changes", func() {
			addition := model.Change{ChangeType: model.Added, HashedFile: index.NewEntry("test1", "h1", "123")}

			err := i.CommitChange("123", addition)
			Expect(err).NotTo(HaveOccurred())

			i, err = index.LoadIndexFile(temp.Name())
			Expect(err).NotTo(HaveOccurred())
			changesAfterAddition := i.CalculateChanges([]model.HashedFile{addition})
			Expect(changesAfterAddition).To(BeEmpty())
		})
	})
})
