package main

// Planner backend: the manuscript-side card proposals that feed the Planner's
// "unplanned" cards. The Planner's own data (lanes, cards, notes) lives in
// planner.json inside the .draftline archive and is edited by the frontend;
// this file only reads the manuscript through the isolated narrative engine
// and hands back proposals with their evidence. Nothing here writes to the
// book.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"draftline/internal/narrative"
	"draftline/internal/plotwalker"
	"draftline/internal/types"
)

// PlannerDetectedCard is one development the narrative engine found in the
// text, positioned by chapter and scene, ready to be shown as an unplanned
// card the writer can adopt or dismiss. Evidence is the sentence the card
// was nominated from.
type PlannerDetectedCard struct {
	ID            string                  `json:"id"`
	Title         string                  `json:"title"`
	Synopsis      string                  `json:"synopsis"`
	ChapterID     string                  `json:"chapter_id"`
	Scene         int                     `json:"scene"`
	Who           []string                `json:"who"`
	Evidence      []types.PlannerEvidence `json:"evidence"`
	Kind          string                  `json:"kind"`
	Status        string                  `json:"status"`
	Support       string                  `json:"support"`
	DiscourseMode string                  `json:"discourse_mode"`
	SelectionRule string                  `json:"selection_rule"`
}

type PlannerDetection struct {
	Revision    string                `json:"revision"`
	SourceID    string                `json:"source_id"`
	Cards       []PlannerDetectedCard `json:"cards"`
	Limitations []string              `json:"limitations"`
	Error       string                `json:"error,omitempty"`
}

// PlannerDetectCards runs the narrative engine's card projection over the
// whole body and maps its proposals onto Planner positions. Proposals are
// candidates, never verified developments; the frontend shows them dashed
// and the writer decides.
func (a *App) PlannerDetectCards(book types.BookData) (result PlannerDetection) {
	defer func() {
		if r := recover(); r != nil {
			result = plannerDetectionError(fmt.Errorf("card detection failed: %v", r))
		}
	}()
	if len(book.Body) == 0 {
		return PlannerDetection{Cards: []PlannerDetectedCard{}, Limitations: []string{}}
	}
	sourceID, err := plotwalker.SourceID(book)
	if err != nil {
		return PlannerDetection{Cards: []PlannerDetectedCard{}, Limitations: []string{}, Error: err.Error()}
	}
	doc, err := narrative.BodyDocument(book, 0, len(book.Body))
	if err != nil {
		return plannerDetectionError(err)
	}
	projection, err := narrative.AnalyzeStoryCards(narrative.CardInput{
		SourceID: sourceID,
		Kind:     narrative.CardManuscript,
		Document: doc,
		Entities: narrative.BookEntities(book),
	})
	if err != nil {
		return plannerDetectionError(err)
	}
	cards := make([]PlannerDetectedCard, 0, len(projection.Cards))
	occurrences := map[string]int{}
	for _, c := range projection.Cards {
		chapterID, scene, err := plannerPosition(book, c.Position.BlockID, c.Position.SceneID)
		if err != nil {
			return PlannerDetection{SourceID: sourceID, Revision: projection.Revision, Cards: []PlannerDetectedCard{}, Limitations: projection.Limitations, Error: err.Error()}
		}
		evidence, err := plannerEvidence(book, sourceID, doc, c.Evidence)
		if err != nil {
			return PlannerDetection{SourceID: sourceID, Revision: projection.Revision, Cards: []PlannerDetectedCard{}, Limitations: projection.Limitations, Error: err.Error()}
		}
		who := make([]string, 0, len(c.Entities))
		for _, e := range c.Entities {
			who = append(who, e.EntityID)
		}
		identity := plannerCandidateIdentity(chapterID, c.Description)
		occurrence := occurrences[identity]
		occurrences[identity]++
		cards = append(cards, PlannerDetectedCard{
			ID:            plannerCandidateIdentity(identity, strconv.Itoa(occurrence)),
			Title:         plotwalker.Title(c.Title),
			Synopsis:      c.Description,
			ChapterID:     chapterID,
			Scene:         scene,
			Who:           who,
			Evidence:      evidence,
			Kind:          "development",
			Status:        c.Status,
			Support:       c.Support,
			DiscourseMode: c.DiscourseMode,
			SelectionRule: c.Selection,
		})
	}
	return PlannerDetection{Revision: projection.Revision, SourceID: sourceID, Cards: cards, Limitations: projection.Limitations}
}

func plannerDetectionError(err error) PlannerDetection {
	return PlannerDetection{Cards: []PlannerDetectedCard{}, Limitations: []string{}, Error: err.Error()}
}

// plannerPosition resolves the engine's block and scene IDs
// ("<chapterKey>/block/N", "<chapterKey>/scene/S") to the chapter's stable ID
// and a 1-based scene number. Chapters that predate stable IDs are keyed
// "body/NNN" by the engine and resolved by index.
func plannerPosition(book types.BookData, blockID, sceneID string) (string, int, error) {
	chapterKey := blockID
	if i := strings.LastIndex(chapterKey, "/block/"); i >= 0 {
		chapterKey = chapterKey[:i]
	} else {
		return "", 0, fmt.Errorf("invalid Planner block position %q", blockID)
	}
	i := strings.LastIndex(sceneID, "/scene/")
	if i < 0 || sceneID[:i] != chapterKey {
		return "", 0, fmt.Errorf("Planner block and scene positions disagree")
	}
	n, err := strconv.Atoi(sceneID[i+len("/scene/"):])
	if err != nil || n < 0 {
		return "", 0, fmt.Errorf("invalid Planner scene position %q", sceneID)
	}
	scene := n + 1
	if strings.HasPrefix(chapterKey, "body/") {
		if index, err := strconv.Atoi(strings.TrimPrefix(chapterKey, "body/")); err == nil && index >= 0 && index < len(book.Body) && book.Body[index].ID != "" {
			return book.Body[index].ID, scene, nil
		}
	}
	for _, chapter := range book.Body {
		if chapter.ID == chapterKey {
			return chapterKey, scene, nil
		}
	}
	return "", 0, fmt.Errorf("Planner proposal references unknown chapter %q", chapterKey)
}

func plannerEvidence(book types.BookData, sourceID string, document narrative.Document, anchors []narrative.Anchor) ([]types.PlannerEvidence, error) {
	scenes := make(map[string]string, len(document.Blocks))
	for _, block := range document.Blocks {
		scenes[block.ID] = block.Scene
	}
	result := make([]types.PlannerEvidence, 0, len(anchors))
	for _, anchor := range anchors {
		sceneID, ok := scenes[anchor.BlockID]
		if !ok {
			return nil, fmt.Errorf("Planner evidence references unknown block %q", anchor.BlockID)
		}
		chapterID, scene, err := plannerPosition(book, anchor.BlockID, sceneID)
		if err != nil {
			return nil, err
		}
		result = append(result, types.PlannerEvidence{SourceID: sourceID, Revision: anchor.Revision, ChapterID: chapterID,
			Scene: scene, BlockID: anchor.BlockID, Start: anchor.Start, End: anchor.End, Quote: anchor.Quote})
	}
	return result, nil
}

func plannerCandidateIdentity(parts ...string) string {
	data, _ := json.Marshal(parts)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:12])
}
