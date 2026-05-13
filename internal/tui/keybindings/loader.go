// MIGRATION TARGET: github.com/hylla-org/lykta
package keybindings

import (
	"encoding/json"
	"fmt"
	"os"
)

// TODO(KEYBIND-R3): refresh embedded stil baseline bytes when stil-solid v<X> publishes
//
// stilBaselineTillsynJSON embeds the 4 Tillsyn commands from the stil baseline in the
// nested product_extensions schema. This copy is pinned at build time and must be kept
// in sync with the stil upstream manually until KEYBIND-R3 lands.
var stilBaselineTillsynJSON = []byte(`{
  "product_extensions": {
    "tillsyn": {
      "commands": [
        {"id":"new-drop",      "keys":["Space","n"], "description":"New drop in current project."},
        {"id":"complete-drop", "keys":["Space","c"], "description":"Mark drop complete."},
        {"id":"handoff",       "command":"handoff",  "description":"Open handoff dialog for current drop."},
        {"id":"comment",       "command":"comment",  "description":"Add a comment thread to current drop."}
      ]
    }
  }
}`)

// Command is a parsed command-palette entry from stil's product_extensions.
type Command struct {
	ID          string   `json:"id"`
	Keys        []string `json:"keys,omitempty"`
	CommandName string   `json:"command,omitempty"`
	Description string   `json:"description"`
}

// Bindings holds the merged command set after loading baseline and optional local overrides.
type Bindings struct {
	Commands []Command
}

// bindingsFile is the internal JSON schema used for both the embedded baseline bytes and the
// Tillsyn-local .tillsyn/bindings.json file. Extra top-level fields (schema_version, name,
// description, extends) are silently ignored by the decoder.
type bindingsFile struct {
	ProductExtensions struct {
		Tillsyn struct {
			Commands []Command `json:"commands"`
		} `json:"tillsyn"`
	} `json:"product_extensions"`
}

// LoadBindings parses baselineJSON (nested-schema bytes) and optionally ID-merges commands
// from the file at localPath. Both inputs use the same bindingsFile schema.
//
// Absent local file (localPath == "" or file does not exist) is graceful baseline-only
// fallback — not an error. Malformed local file returns an error.
//
// Returns 4 commands when no local file is present; 9 commands when the canonical local
// file (5 non-colliding entries) is merged. Local wins on ID collision.
func LoadBindings(baselineJSON []byte, localPath string) (Bindings, error) {
	var base bindingsFile
	if err := json.Unmarshal(baselineJSON, &base); err != nil {
		return Bindings{}, fmt.Errorf("keybindings: parse baseline JSON: %w", err)
	}

	baseline := base.ProductExtensions.Tillsyn.Commands

	if localPath == "" {
		return Bindings{Commands: baseline}, nil
	}

	f, err := os.Open(localPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Bindings{Commands: baseline}, nil
		}
		return Bindings{}, fmt.Errorf("keybindings: open local bindings %q: %w", localPath, err)
	}
	defer f.Close()

	var local bindingsFile
	if err := json.NewDecoder(f).Decode(&local); err != nil {
		return Bindings{}, fmt.Errorf("keybindings: parse local bindings %q: %w", localPath, err)
	}

	merged := idMerge(baseline, local.ProductExtensions.Tillsyn.Commands)
	return Bindings{Commands: merged}, nil
}

// DefaultBaselineJSON returns the embedded stil baseline bytes for the Tillsyn product
// extension. Callers pass this to LoadBindings as the baselineJSON argument.
func DefaultBaselineJSON() []byte {
	return stilBaselineTillsynJSON
}

// idMerge performs an ID-based deep merge of base and overrides. For each override command
// whose ID matches a base command, the override replaces the base entry. New IDs are appended.
// Local (override) wins on collision.
func idMerge(base, overrides []Command) []Command {
	// Build index of base commands by ID.
	idx := make(map[string]int, len(base))
	result := make([]Command, len(base))
	copy(result, base)
	for i, c := range result {
		idx[c.ID] = i
	}

	for _, ov := range overrides {
		if i, found := idx[ov.ID]; found {
			result[i] = ov
		} else {
			result = append(result, ov)
			idx[ov.ID] = len(result) - 1
		}
	}

	return result
}
