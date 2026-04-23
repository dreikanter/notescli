package note

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// idFile represents the id.json file that tracks the last allocated note ID.
type idFile struct {
	LastID int `json:"last_id"`
}

// readID reads the current last_id from id.json in the store root.
func readID(root string) (idFile, error) {
	data, err := os.ReadFile(filepath.Join(root, "id.json"))
	if err != nil {
		return idFile{}, fmt.Errorf("cannot read id.json: %w", err)
	}
	var idf idFile
	if err := json.Unmarshal(data, &idf); err != nil {
		return idFile{}, fmt.Errorf("cannot parse id.json: %w", err)
	}
	return idf, nil
}

// NextID reads id.json, increments last_id, writes it back, and returns the new ID.
// The read-modify-write is serialized across processes via an exclusive flock on
// the store root directory, so concurrent callers cannot duplicate IDs.
func NextID(root string) (int, error) {
	unlock, err := lockStoreRoot(root)
	if err != nil {
		return 0, err
	}
	defer unlock()

	idf, err := readID(root)
	if err != nil {
		return 0, err
	}
	idf.LastID++
	if err := writeID(root, idf); err != nil {
		return 0, err
	}
	return idf.LastID, nil
}

// writeID atomically writes the idFile to id.json using a temp file + rename.
func writeID(root string, idf idFile) error {
	data, err := json.Marshal(idf)
	if err != nil {
		return fmt.Errorf("cannot marshal id.json: %w", err)
	}

	target := filepath.Join(root, "id.json")
	tmp, err := os.CreateTemp(root, "id-*.json.tmp")
	if err != nil {
		return fmt.Errorf("cannot create temp file for id.json: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("cannot write temp id.json: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("cannot close temp id.json: %w", err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("cannot rename temp id.json: %w", err)
	}
	return nil
}
