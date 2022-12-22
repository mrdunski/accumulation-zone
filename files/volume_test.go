package files

import (
	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	. "github.com/mrdunski/accumulation-zone/gomega"
	"github.com/mrdunski/accumulation-zone/model/mock_model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

type fileMatcher struct {
	fh TreeHashedFile
}

func (f fileMatcher) Match(actual interface{}) (success bool, err error) {
	return f.fh.Equal(actual.(TreeHashedFile)), nil
}

func (f fileMatcher) FailureMessage(actual interface{}) (message string) {
	fileHandle := actual.(TreeHashedFile)
	return fmt.Sprintf("Expected {%s, %s} to be equal {%s, %s}", fileHandle.path, fileHandle.treeHash, f.fh.path, f.fh.treeHash)
}

func (f fileMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	fileHandle := actual.(TreeHashedFile)
	return fmt.Sprintf("Expected {%s, %s} to be other than {%s, %s}", fileHandle.path, fileHandle.treeHash, f.fh.path, f.fh.treeHash)
}

func fileWith(path, hash string) interface{} {
	return fileMatcher{TreeHashedFile{
		path:     path,
		treeHash: hash,
	}}
}

var _ = Describe("volume.LoadTree", func() {
	_, f, _, _ := runtime.Caller(0)
	dirPath := filepath.Dir(f)
	testDir := filepath.Join(dirPath, "test")
	When("valid path provided", func() {
		validPath := filepath.Join(testDir, "valid")
		loader := NewLoader(validPath)
		It("loads all files", func() {
			tree, err := loader.LoadTree()
			Expect(err).NotTo(HaveOccurred())
			Expect(tree).To(ContainElements(
				fileWith("dir/testfile1", "05e8fdb3598f91bcc3ce41a196e587b4592c8cdfc371c217274bfda2d24b1b4e"),
				fileWith("dir/testfile2", "26637da1bd793f9011a3d304372a9ec44e36cc677d2bbfba32a2f31f912358fe"),
				fileWith("empty-file", ""),
			))
			Expect(tree).To(HaveLen(3))
		})
	})

	When("file is broken", func() {
		brokenPath := filepath.Join(testDir, "broken")
		loader := NewLoader(brokenPath)
		It("returns an error", func() {
			_, err := loader.LoadTree()
			Expect(err).To(HaveOccurred())
		})
	})

	When("dir is missing", func() {
		missingPath := filepath.Join(testDir, "missing")
		loader := NewLoader(missingPath)
		It("returns an error", func() {
			_, err := loader.LoadTree()
			Expect(err).To(HaveOccurred())
		})
	})

	When("path is file", func() {
		filePath := filepath.Join(testDir, "valid", "empty-file")
		loader := NewLoader(filePath)
		It("returns an error", func() {
			_, err := loader.LoadTree()
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("volume.LoadFile", func() {
	When("valid file is given", func() {
		var file TreeHashedFile

		BeforeEach(func() {
			_, rtFile, _, _ := runtime.Caller(0)
			dirPath := filepath.Dir(rtFile)
			dir := filepath.Join(dirPath, "test", "valid", "dir")
			loader := NewLoader(dir)
			f, err := loader.LoadFile("testfile1")
			Expect(err).NotTo(HaveOccurred())
			file = f
		})

		It("loads file content", func() {
			content, err := file.Content()
			Expect(err).NotTo(HaveOccurred())
			b := new(strings.Builder)
			_, err = io.Copy(b, content)
			Expect(err).NotTo(HaveOccurred())
			Expect(b.String()).To(Equal("test data 1"))
		})

		It("calculates file hash", func() {
			Expect(file.Hash()).To(Equal("05e8fdb3598f91bcc3ce41a196e587b4592c8cdfc371c217274bfda2d24b1b4e"))
		})
	})

})

var _ = Describe("volume.save", func() {
	var testDir string
	BeforeEach(func() {
		temp, err := os.MkdirTemp(os.TempDir(), "acc-vol-*")
		if !Expect(err).NotTo(HaveOccurred()) {
			return
		}

		testDir = temp
	})

	It("saves content into new file", func() {
		file := mock_model.NewMockFileWithContent(gomock.NewController(GinkgoT()))
		file.EXPECT().Path().AnyTimes().Return("file.txt")
		file.EXPECT().Content().Return(io.NopCloser(strings.NewReader("test-content")), nil)

		loader := NewLoader(testDir)
		err := loader.Save(file)

		Expect(err).ToNot(HaveOccurred())

		actualContent, err := os.ReadFile(path.Join(testDir, "file.txt"))

		Expect(err).ToNot(HaveOccurred())
		Expect(string(actualContent)).To(Equal("test-content"))
	})

	It("saves content into existing file", func() {
		_ = os.WriteFile(path.Join(testDir, "existing.txt"), []byte("???"), 0666)

		file := mock_model.NewMockFileWithContent(gomock.NewController(GinkgoT()))
		file.EXPECT().Path().AnyTimes().Return("existing.txt")
		file.EXPECT().Content().Return(io.NopCloser(strings.NewReader("test-content")), nil)

		loader := NewLoader(testDir)
		err := loader.Save(file)

		Expect(err).ToNot(HaveOccurred())

		actualContent, err := os.ReadFile(path.Join(testDir, "existing.txt"))

		Expect(err).ToNot(HaveOccurred())
		Expect(string(actualContent)).To(Equal("test-content"))
	})

	It("handles content err", func() {
		expectedErr := errors.New("content err")

		file := mock_model.NewMockFileWithContent(gomock.NewController(GinkgoT()))
		file.EXPECT().Path().AnyTimes().Return("file.txt")
		file.EXPECT().Content().Return(nil, expectedErr)

		loader := NewLoader(testDir)
		err := loader.Save(file)

		Expect(err).To(WrapError(expectedErr))
	})
})
