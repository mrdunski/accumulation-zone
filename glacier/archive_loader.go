package glacier

import (
	"github.com/mrdunski/accumulation-zone/model"
	"io"
)

type archiveLoader struct {
	model.IdentifiableHashedFile
	openContent func() (io.ReadCloser, error)
}

func (l archiveLoader) Content() (io.ReadCloser, error) {
	return l.openContent()
}
