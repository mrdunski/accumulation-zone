package model

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HashedFiles", func() {
	It("should handle adding and finding files", func() {
		hf := HashedFiles{}
		hf.Replace(file{
			path: "p1",
			hash: "h1",
		})

		Expect(hf.HasFile("p1", "h1")).To(BeTrue())
		Expect(hf.HasFile("p1", "h2")).To(BeFalse())
		Expect(hf.HasFile("p2", "h1")).To(BeFalse())
	})
})

var _ = Describe("IdentifiableHashedFiles", func() {
	It("keeps newest hash files", func() {
		filesWithDuplicte := []file{
			{
				path: "p1",
				hash: "h1",
			},
			{
				path: "p1",
				hash: "h2",
			},
			{
				path: "p2",
				hash: "h3",
			},
		}
		set := NewIdentifiableHashedFiles(filesWithDuplicte)

		Expect(set.HasFile("p1", "h2")).To(BeTrue())
		Expect(set.HasFile("p1", "h1")).To(BeFalse())
		Expect(set.HasFile("p2", "h3")).To(BeTrue())
	})
})
