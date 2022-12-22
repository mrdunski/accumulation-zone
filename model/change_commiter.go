//go:generate mockgen -destination=mock_model/change_commiter.go . ChangeCommitter
package model

type ChangeCommitter interface {
	CommitAdd(changeId string, changed HashedFile) error
	CommitDelete(changeId string, changed HashedFile) error
}
