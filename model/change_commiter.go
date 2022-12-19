//go:generate mockgen -destination=mock_model/change_commiter.go . ChangeCommitter
package model

type ChangeCommitter interface {
	CommitChange(changeId string, change Change) error
}
