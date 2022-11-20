package model

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HashedFiles", func() {
	It("should handle adding and finding files", func() {
		hf := HashedFiles{}
		hf.Add(hashFile{
			path: "p1",
			hash: "h1",
		})

		Expect(hf.HasFile("p1", "h1")).To(BeTrue())
		Expect(hf.HasFile("p1", "h2")).To(BeFalse())
		Expect(hf.HasFile("p2", "h1")).To(BeFalse())
	})
})

var _ = Describe("AsHashedFiles", func() {
	It("converts slice of anything to []HashFile", func() {
		t := []hashFile{{
			path: "p1",
			hash: "h1",
		},
			{
				path: "p2",
				hash: "h2",
			},
		}

		hashedFiles := AsHashedFiles(t)
		for _, item := range t {
			Expect(hashedFiles).To(ContainElement(item))
		}
	})
})
