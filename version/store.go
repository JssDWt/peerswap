package version

import "fmt"

var (
	ErrDoesNotExist = fmt.Errorf("does not exist")
)

type VersionStore interface {
	GetVersion() (string, error)
	SetVersion(version string) error
}
