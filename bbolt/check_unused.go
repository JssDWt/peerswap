package bbolt

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/elementsproject/peerswap/log"
	"go.etcd.io/bbolt"
)

var ErrExistingSwaps = fmt.Errorf("active swaps in deprecated bbolt database")

func CheckUnused(datadir string) error {
	bboltDbFile := filepath.Join(datadir, "swaps")
	_, err := os.Stat(bboltDbFile)
	if os.IsNotExist(err) {
		return nil
	}

	bboltdb, err := bbolt.Open(bboltDbFile, 0700, nil)
	if err != nil {
		log.Infof(
			"Failed to open deprecated bbolt database '%s' to check for active swaps. "+
				"Assuming no active swaps and it's safe to ignore this deprecated database. "+
				"error: %v",
			bboltDbFile,
			err)
		return nil
	}

	bboltSwapStore, err := NewBBoltSwapStore(bboltdb)
	if err != nil {
		log.Infof(
			"Failed to create deprecated bbolt swap store to check for active swaps. "+
				"Assuming no active swaps and it's safe to ignore this deprecated database. "+
				"error: %v",
			err)
		return nil
	}

	active, err := bboltSwapStore.HasActiveSwaps()
	if err != nil {
		log.Infof(
			"Failed to check for active swaps in deprecated bbolt database. "+
				"Assuming no active swaps and it's safe to ignore this database. "+
				"error: %v",
			err)
		return nil
	}

	if !active {
		return nil
	}

	versionStore, err := NewBBoltVersionStore(bboltdb)
	if err != nil {
		log.Infof(
			"Found active swaps in deprecated bbolt database with unknown version. " +
				"Run the previous version of peerswap to resolve these swaps before upgrading.")
		return ErrExistingSwaps
	}

	version, err := versionStore.GetVersion()
	if err != nil {
		log.Infof(
			"Found active swaps in deprecated bbolt database with unknown version. " +
				"Run the previous version of peerswap to resolve these swaps before upgrading.")
		return ErrExistingSwaps
	}

	log.Infof(
		"Found active swaps in deprecated bbolt database with version '%s'. "+
			"Run that version of peerswap to resolve these swaps before upgrading.",
		version)
	return ErrExistingSwaps
}
