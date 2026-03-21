package note

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// IDFile represents the id.json file that tracks the last allocated note ID.
type IDFile struct {
	LastID int `json:"last_id"`
}

// ReadID reads the current last_id from id.json in the store root.
func ReadID(root string) (IDFile, error) {
	data, err := os.ReadFile(filepath.Join(root, "id.json"))
	if err != nil {
		return IDFile{}, fmt.Errorf("cannot read id.json: %w", err)
	}
	var idf IDFile
	if err := json.Unmarshal(data, &idf); err != nil {
		return IDFile{}, fmt.Errorf("cannot parse id.json: %w", err)
	}
	return idf, nil
}

// NextID reads id.json, increments last_id, writes it back atomically, and returns the new ID.
func NextID(root string) (int, error) {
	idf, err := ReadID(root)
	if err != nil {
		return 0, err
	}
	idf.LastID++
	if err := WriteID(root, idf); err != nil {
		return 0, err
	}
	return idf.LastID, nil
}

// WriteID atomically writes the IDFile to id.json using a temp file + rename.
func WriteID(root string, idf IDFile) error {
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
