// Package sync handles bidirectional synchronization between the model and draw.io files.
package sync

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/docToolchain/Bauteinsicht/internal/drawio"
	"github.com/docToolchain/Bauteinsicht/internal/model"
)

// SyncState stores the state after each successful sync.
type SyncState struct {
	Timestamp     string                  `json:"timestamp"`
	ModelHash     string                  `json:"model_hash"`
	DrawioHash    string                  `json:"drawio_hash"`
	Elements      map[string]ElementState `json:"elements"`
	Relationships []RelationshipState     `json:"relationships"`
}

// ElementState captures an element's synced values.
type ElementState struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Technology  string `json:"technology,omitempty"`
	Kind        string `json:"kind"`
}

// RelationshipState captures a relationship's synced values.
type RelationshipState struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Index int    `json:"index"`
	Label string `json:"label,omitempty"`
	Kind  string `json:"kind,omitempty"`
}

// LoadState reads a SyncState from the given path.
// If the file does not exist, an empty SyncState is returned (first-sync scenario).
func LoadState(path string) (*SyncState, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path derived from model location
	if err != nil {
		if os.IsNotExist(err) {
			return &SyncState{
				Elements:      make(map[string]ElementState),
				Relationships: []RelationshipState{},
			}, nil
		}
		return nil, fmt.Errorf("LoadState %q: %w", path, err)
	}

	// Treat a zero-byte file as empty/missing state (e.g. truncated write).
	if len(data) == 0 {
		return &SyncState{
			Elements:      make(map[string]ElementState),
			Relationships: []RelationshipState{},
		}, nil
	}

	var state SyncState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("LoadState %q: %w", path, err)
	}
	if state.Elements == nil {
		state.Elements = make(map[string]ElementState)
	}
	if state.Relationships == nil {
		state.Relationships = []RelationshipState{}
	}
	return &state, nil
}

// SaveState atomically writes state to path using a temp file + rename.
func SaveState(path string, state *SyncState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("SaveState marshal: %w", err)
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".bausteinsicht-sync-tmp-*")
	if err != nil {
		return fmt.Errorf("SaveState create temp: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("SaveState write: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("SaveState close: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("SaveState rename: %w", err)
	}
	return nil
}

// ComputeHash reads the file at path and returns a "sha256:<hex>" fingerprint.
func ComputeHash(path string) (string, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- path derived from model location
	if err != nil {
		return "", fmt.Errorf("ComputeHash %q: %w", path, err)
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum), nil
}

// BuildState creates a SyncState snapshot from the current model and draw.io document.
// doc is accepted but not inspected (draw.io element data comes from the model for now).
func BuildState(m *model.BausteinsichtModel, _ *drawio.Document, modelPath, drawioPath string) (*SyncState, error) {
	modelHash, err := ComputeHash(modelPath)
	if err != nil {
		return nil, fmt.Errorf("BuildState model hash: %w", err)
	}

	drawioHash, err := ComputeHash(drawioPath)
	if err != nil {
		return nil, fmt.Errorf("BuildState drawio hash: %w", err)
	}

	flat := model.FlattenElements(m)
	elements := make(map[string]ElementState, len(flat))
	for id, elem := range flat {
		elements[id] = ElementState{
			Title:       elem.Title,
			Description: elem.Description,
			Technology:  elem.Technology,
			Kind:        elem.Kind,
		}
	}

	rels := make([]RelationshipState, 0, len(m.Relationships))
	for i, r := range m.Relationships {
		rels = append(rels, RelationshipState{
			From:  r.From,
			To:    r.To,
			Index: i,
			Label: r.Label,
			Kind:  r.Kind,
		})
	}

	return &SyncState{
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		ModelHash:     modelHash,
		DrawioHash:    drawioHash,
		Elements:      elements,
		Relationships: rels,
	}, nil
}
