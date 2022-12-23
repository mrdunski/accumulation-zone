package model

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"io"
	"strings"
)

type file struct {
	path string
	hash string
}

func (h file) Path() string {
	return h.path
}

func (h file) Hash() string {
	return h.hash
}

func (h file) ChangeId() string {
	return strings.Join([]string{h.path, h.hash}, ",")
}

func (h file) Content() (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

var _ = DescribeTable("Format message", func(stringer fmt.Stringer, expectedResult string) {
	val := fmt.Sprintf("%v", stringer)
	Expect(val).To(Equal(expectedResult))
},
	Entry("FileAdded", FileAdded{FileWithContent: file{path: "abc", hash: "h"}}, "{added: {abc h}}"),
	Entry("FileAdded with nil", FileAdded{}, "{added: ?}"),
	Entry("FileDeleted", FileDeleted{IdentifiableHashedFile: file{path: "abc", hash: "h"}}, "{deleted: {abc h | abc,h}}"),
	Entry("FileDeleted with nil", FileDeleted{}, "{deleted: ?}"),
	Entry("Changes", &Changes{}, "{added: [], deleted: []}"),
	Entry("Changes", &Changes{Additions: []FileAdded{{}}, Deletions: []FileDeleted{{}}}, "{added: [{added: ?}], deleted: [{deleted: ?}]}"),
)

var _ = Describe("Changes", func() {
	It("appends other changes", func() {
		changes1 := Changes{
			Additions: []FileAdded{{FileWithContent: file{path: "1"}}},
			Deletions: []FileDeleted{{IdentifiableHashedFile: file{path: "2"}}},
		}
		changes2 := Changes{
			Additions: []FileAdded{{FileWithContent: file{path: "3"}}},
			Deletions: []FileDeleted{{IdentifiableHashedFile: file{path: "4"}}, {IdentifiableHashedFile: file{path: "5"}}},
		}

		changes1.Append(changes2)

		Expect(changes1.Additions).To(ConsistOf(FileAdded{FileWithContent: file{path: "1"}}, FileAdded{FileWithContent: file{path: "3"}}))
		Expect(changes1.Deletions).To(ConsistOf(FileDeleted{IdentifiableHashedFile: file{path: "2"}}, FileDeleted{IdentifiableHashedFile: file{path: "4"}}, FileDeleted{IdentifiableHashedFile: file{path: "5"}}))
	})
})
