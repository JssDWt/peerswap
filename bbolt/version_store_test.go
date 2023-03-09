package bbolt

import (
	"path/filepath"
	"testing"

	"github.com/elementsproject/peerswap/version"
	"github.com/stretchr/testify/assert"
	"go.etcd.io/bbolt"
)

func Test_VersionStore(t *testing.T) {
	// db
	boltdb, err := bbolt.Open(filepath.Join(t.TempDir(), "swaps"), 0700, nil)
	if err != nil {
		t.Fatal(err)
	}

	versionStore, err := NewBBoltVersionStore(boltdb)
	if err != nil {
		t.Fatal(err)
	}

	newVersion := "v0.2.0-beta"

	oldVersion, err := versionStore.GetVersion()
	if err != version.ErrDoesNotExist && err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "", oldVersion)

	err = versionStore.SetVersion(newVersion)
	if err != nil {
		t.Fatal(err)
	}

	setVersion, err := versionStore.GetVersion()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, newVersion, setVersion)

	boltdb.Close()
}
