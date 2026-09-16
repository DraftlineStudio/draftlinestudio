package book

import (
	"fmt"

	"draftline/internal/types"
)

const plannerDataVersion = 1

// preparePlannerData validates the archive contract and normalizes required
// collections for the frontend. Unknown versions are refused to protect data
// written by a newer Draftline release from being overwritten by this one.
// A card whose stored people's names do not line up with its people (a
// hand-edited or partly migrated card) loses the names rather than naming
// its people by position from the wrong list.
func preparePlannerData(planner types.PlannerData) (types.PlannerData, error) {
	if planner.Version != plannerDataVersion {
		return types.PlannerData{}, fmt.Errorf("planner data version %d is not supported", planner.Version)
	}
	for i, card := range planner.Cards {
		if card.WhoNames != nil && len(card.WhoNames) != len(card.Who) {
			planner.Cards[i].WhoNames = nil
		}
	}
	if planner.Lanes == nil {
		planner.Lanes = []types.PlannerLane{}
	}
	if planner.Cards == nil {
		planner.Cards = []types.PlannerCard{}
	}
	if planner.Notes == nil {
		planner.Notes = []types.PlannerNote{}
	}
	return planner, nil
}
