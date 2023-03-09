package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockVersionStore struct {
	version string
}

func (vs *mockVersionStore) GetVersion() (string, error) {
	if vs.version == "" {
		return "", ErrDoesNotExist
	}

	return vs.version, nil
}

func (vs *mockVersionStore) SetVersion(version string) error {
	vs.version = version
	return nil
}

func Test_VersionService(t *testing.T) {
	mockVersionStore := &mockVersionStore{}
	versionService, err := NewVersionService(mockVersionStore)
	if err != nil {
		t.Fatal(err)
	}

	activeSwaps := &MockActiveSwaps{true}
	err = versionService.SafeUpgrade(activeSwaps)
	assert.Error(t, err)
	if _, ok := err.(ActiveSwapsError); !ok {
		t.Fatalf("Error not of type ActiveSwapsError")
	}

	activeSwaps = &MockActiveSwaps{false}
	err = versionService.SafeUpgrade(activeSwaps)
	assert.NoError(t, err)
}

type MockActiveSwaps struct {
	hasActiveSwaps bool
}

func (m *MockActiveSwaps) HasActiveSwaps() (bool, error) {
	return m.hasActiveSwaps, nil
}
