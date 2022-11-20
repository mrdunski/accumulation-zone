package model

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type hashFile struct {
	path string
	hash string
}

func (h hashFile) Path() string {
	return h.path
}

func (h hashFile) Hash() string {
	return h.hash
}

var _ = Describe("Change", func() {
	Describe("Stringer", func() {
		It("formats data well", func() {
			val := fmt.Sprintf("%v", Change{ChangeType: Added, HashedFile: hashFile{path: "abc", hash: "h"}})
			Expect(val).To(Equal("{added: {abc h}}"))
		})

		It("handles nil", func() {
			val := fmt.Sprintf("%v", Change{ChangeType: Deleted})
			Expect(val).To(Equal("{deleted: ?}"))
		})
	})
})
