//go:build !js

package game

import (
	"fmt"
	"os"
)

const saveSlot = "save.json"
const scoreSlot = "scores.json"

// fileStore persists slots as files.
type fileStore struct{}

func defaultStore() slotStore { return fileStore{} }

func (fileStore) read(slot string) ([]byte, error) {
	data, err := os.ReadFile(slot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read save: %w", err)
	}
	return data, nil
}

func (fileStore) write(slot string, data []byte) error {
	return os.WriteFile(slot, data, 0600)
}

func (fileStore) delete(slot string) error {
	if err := os.Remove(slot); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (fileStore) exists(slot string) bool {
	_, err := os.Stat(slot)
	return err == nil
}
