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
