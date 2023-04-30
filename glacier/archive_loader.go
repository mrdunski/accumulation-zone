package glacier

import (
	"io"

	"github.com/mrdunski/accumulation-zone/model"
)

type archiveLoader struct {
	model.IdentifiableHashedFile
	getSize     func() (int64, error)
	openContent func() (io.ReadCloser, error)
}

func (l archiveLoader) Content() (io.ReadCloser, error) {
	return l.openContent()
}

func (l archiveLoader) Size() (int64, error) {
	return l.getSize()
}
