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
)
