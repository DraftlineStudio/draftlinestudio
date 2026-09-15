package book

import (
	"fmt"

	"draftline/internal/types"
)

const plannerDataVersion = 1

// preparePlannerData validates the archive contract and normalizes required
// collections for the frontend. Unknown versions are refused to protect data
// written by a newer Draftline release from being overwritten by this one.
func preparePlannerData(planner types.PlannerData) (types.PlannerData, error) {
	if planner.Version != plannerDataVersion {
		return types.PlannerData{}, fmt.Errorf("planner data version %d is not supported", planner.Version)
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
