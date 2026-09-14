// Package narrative is an experimental evidence-first analysis layer. It is
// deliberately not connected to fingerprint.Build or the production timeline.
package narrative

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Offsets are UTF-8 byte offsets in the decoded text of ONE block, never in
// HTML or in a chapter flattened across paragraphs/scene separators.
type Block struct {
	ID    string `json:"id"`
	Scene string `json:"scene"`
	Text  string `json:"text"`
	Mode  string `json:"mode"`
}

type Document struct {
	Revision string  `json:"revision"`
	Blocks   []Block `json:"blocks"`
}

func NewDocument(blocks []Block) Document {
	copyBlocks := append([]Block(nil), blocks...)
	for i := range copyBlocks {
		if copyBlocks[i].Mode == "" {
			copyBlocks[i].Mode = "current_narration"
		}
	}
	data, _ := json.Marshal(copyBlocks)
	return Document{Revision: id(string(data)), Blocks: copyBlocks}
}

type Anchor struct {
	Revision string `json:"revision"`
	BlockID  string `json:"block_id"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Quote    string `json:"quote"`
}

func (d Document) Anchor(block int, start, end int) Anchor {
	b := d.Blocks[block]
	return Anchor{d.Revision, b.ID, start, end, b.Text[start:end]}
}

func (d Document) Validate() error {
	if d.Revision != NewDocument(d.Blocks).Revision {
		return fmt.Errorf("document revision does not match its blocks")
	}
	seen := map[string]bool{}
	closedScenes := map[string]bool{}
	previousScene := ""
	for _, b := range d.Blocks {
		if b.ID == "" || b.Scene == "" || seen[b.ID] {
			return fmt.Errorf("missing or duplicate block identity: %q", b.ID)
		}
		seen[b.ID] = true
		if b.Scene != previousScene {
			if closedScenes[b.Scene] {
				return fmt.Errorf("non-contiguous scene %s", b.Scene)
			}
			closedScenes[previousScene] = true
			previousScene = b.Scene
		}
	}
	return nil
}

func (d Document) Position(a Anchor) (int, error) {
	if a.Revision != d.Revision {
		return 0, fmt.Errorf("stale source revision")
	}
	for i, b := range d.Blocks {
		if b.ID != a.BlockID {
			continue
		}
		if a.Start < 0 || a.End <= a.Start || a.End > len(b.Text) || b.Text[a.Start:a.End] != a.Quote {
			return 0, fmt.Errorf("invalid source span in %s", b.ID)
		}
		return i, nil
	}
	return 0, fmt.Errorf("unknown block %q", a.BlockID)
}

type Person struct {
	ID    string
	Names []string
}

// Observation is a clause-bound claim, not a state transition. In particular,
// "present" does not assert an entrance and "absent" does not invent an exit.
type Observation struct {
	ID              string   `json:"id"`
	Kind            string   `json:"kind"`
	Subject         string   `json:"subject"`
	Place           string   `json:"place,omitempty"`
	Scene           string   `json:"scene"`
	Mode            string   `json:"mode"`
	Source          Anchor   `json:"source"`
	SubjectSource   Anchor   `json:"subject_source"`
	Rule            string   `json:"rule"`
	PredicateSource *Anchor  `json:"predicate_source,omitempty"`
	PlaceSource     *Anchor  `json:"place_source,omitempty"`
	SpatialRelation string   `json:"spatial_relation,omitempty"`
	Antecedents     []Anchor `json:"antecedents,omitempty"`
	BindingStatus   string   `json:"binding_status,omitempty"`
	ClauseComplete  bool     `json:"clause_complete"`
}

type Inspection struct {
	Kind           string   `json:"kind"`
	Subject        string   `json:"subject"`
	Place          string   `json:"place"`
	Detail         string   `json:"detail"`
	Premises       []string `json:"premises"`
	Sources        []Anchor `json:"sources"`
	SearchFrom     Anchor   `json:"search_from"`
	SearchThrough  Anchor   `json:"search_through"`
	UnparsedBlocks []string `json:"unparsed_blocks"`
	Coverage       string   `json:"coverage"`
}

type Extraction struct {
	Observations   []Observation      `json:"observations"`
	UnparsedBlocks []string           `json:"unparsed_blocks"`
	Movements      []MovementEvidence `json:"movement_candidates,omitempty"`
}

// MovementEvidence preserves a grammatical movement candidate, not an accepted
// plot development or inspection-closing departure. Each participant owns its
// own origin binding; an inferred leader origin is not inherited by followers.
type MovementEvidence struct {
	ID               string                `json:"id"`
	Scene            string                `json:"scene"`
	Source           Anchor                `json:"source"`
	PredicateSources []Anchor              `json:"predicate_sources"`
	RouteSource      *Anchor               `json:"route_source,omitempty"`
	Participants     []MovementParticipant `json:"participants"`
	Complete         bool                  `json:"clause_complete"`
	Rule             string                `json:"rule"`
}

type MovementParticipant struct {
	Subject        string          `json:"subject,omitempty"`
	Role           string          `json:"role"`
	Source         Anchor          `json:"source"`
	IdentityStatus string          `json:"identity_status"`
	Origin         *MovementOrigin `json:"origin,omitempty"`
}

// Explicit origins point into the movement clause. Inferred origins refer to a
// separate observation by ID, never a fabricated span in the movement clause.
type MovementOrigin struct {
	Place   string  `json:"place"`
	Status  string  `json:"status"`
	Source  *Anchor `json:"source,omitempty"`
	Premise string  `json:"premise,omitempty"`
}

func id(parts ...string) string {
	data, _ := json.Marshal(parts)
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:12])
}
