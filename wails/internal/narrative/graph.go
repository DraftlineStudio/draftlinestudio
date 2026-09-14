package narrative

import (
	"fmt"
	"sort"
)

// Development inputs must come from a proposition-binding extractor. This
// package validates provenance and projects links; it does NOT turn proximity,
// common participants or repeated words into supported narrative relations.
type Development struct {
	ID                     string           `json:"id"`
	Label                  string           `json:"label"`
	EstablishedBy          string           `json:"established_by"`
	SupportingObservations []string         `json:"supporting_observations"`
	Changes                []ActivityChange `json:"changes"`
}

type ActivityChange struct {
	ActivityID string `json:"activity_id"`
	Operation  string `json:"operation"` // initiate | continue | complete
	Witness    string `json:"witness"`
}

type Relation struct {
	ID        string   `json:"id"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	Kind      string   `json:"kind"` // continues | branches | joins | fulfills | reports
	Witnesses []string `json:"witnesses"`
	Rule      string   `json:"rule"`
}

type Thread struct {
	ActivityID string   `json:"activity_id"`
	NodeIDs    []string `json:"node_ids"`
	Status     string   `json:"status"` // outcome_not_observed | completed
}

type Graph struct {
	Revision     string        `json:"revision"`
	Observations []Observation `json:"observations"`
	Nodes        []Development `json:"nodes"`
	Relations    []Relation    `json:"relations"`
	Threads      []Thread      `json:"threads"`
}

// BuildGraph is the reduction boundary for the future extractor, not evidence
// that extraction works. Tests with supplied observations exercise this boundary
// only. All node positions come from EstablishedBy, never earliest support.
func BuildGraph(d Document, observations []Observation, nodes []Development, relations []Relation) (Graph, error) {
	if err := d.Validate(); err != nil {
		return Graph{}, err
	}
	g := Graph{Revision: d.Revision, Observations: append([]Observation(nil), observations...), Nodes: append([]Development(nil), nodes...), Relations: append([]Relation(nil), relations...), Threads: []Thread{}}
	byObservation := map[string]Observation{}
	positions := map[string]int{}
	for _, o := range observations {
		if o.ID == "" || o.Rule == "" || o.Mode == "" {
			return Graph{}, fmt.Errorf("observation identity, mode and extraction rule required")
		}
		if _, ok := byObservation[o.ID]; ok {
			return Graph{}, fmt.Errorf("duplicate observation %s", o.ID)
		}
		position, err := d.Position(o.Source)
		if err != nil {
			return Graph{}, err
		}
		byObservation[o.ID] = o
		positions[o.ID] = position
	}
	byNode := map[string]Development{}
	for _, node := range nodes {
		if node.ID == "" || node.Label == "" {
			return Graph{}, fmt.Errorf("node identity and label required")
		}
		if _, ok := byNode[node.ID]; ok {
			return Graph{}, fmt.Errorf("duplicate node %s", node.ID)
		}
		if _, ok := byObservation[node.EstablishedBy]; !ok {
			return Graph{}, fmt.Errorf("node %s has no establishing observation", node.ID)
		}
		for _, support := range node.SupportingObservations {
			if _, ok := byObservation[support]; !ok {
				return Graph{}, fmt.Errorf("unknown support %s", support)
			}
			a, b := byObservation[support], byObservation[node.EstablishedBy]
			if positions[a.ID] > positions[b.ID] || positions[a.ID] == positions[b.ID] && a.Source.End > b.Source.End {
				return Graph{}, fmt.Errorf("node %s uses evidence disclosed after establishment", node.ID)
			}
		}
		for _, change := range node.Changes {
			if change.ActivityID == "" {
				return Graph{}, fmt.Errorf("missing activity identity")
			}
			if change.Witness != node.EstablishedBy && !includes(node.SupportingObservations, change.Witness) {
				return Graph{}, fmt.Errorf("activity change witness not in node support")
			}
			if _, ok := byObservation[change.Witness]; !ok {
				return Graph{}, fmt.Errorf("missing activity change witness")
			}
			if change.Operation != "initiate" && change.Operation != "continue" && change.Operation != "complete" {
				return Graph{}, fmt.Errorf("unsupported activity operation %s", change.Operation)
			}
		}
		byNode[node.ID] = node
	}
	sort.SliceStable(g.Nodes, func(i, j int) bool {
		a, b := byObservation[g.Nodes[i].EstablishedBy], byObservation[g.Nodes[j].EstablishedBy]
		if positions[a.ID] != positions[b.ID] {
			return positions[a.ID] < positions[b.ID]
		}
		if a.Source.Start != b.Source.Start {
			return a.Source.Start < b.Source.Start
		}
		return g.Nodes[i].ID < g.Nodes[j].ID
	})
	seenEdges := map[string]bool{}
	for _, edge := range relations {
		if edge.ID == "" || seenEdges[edge.ID] || edge.Rule == "" || len(edge.Witnesses) == 0 {
			return Graph{}, fmt.Errorf("relation needs unique identity, rule and witnesses")
		}
		seenEdges[edge.ID] = true
		if _, ok := byNode[edge.From]; !ok {
			return Graph{}, fmt.Errorf("unknown relation source")
		}
		if _, ok := byNode[edge.To]; !ok {
			return Graph{}, fmt.Errorf("unknown relation target")
		}
		if edge.From == edge.To {
			return Graph{}, fmt.Errorf("self relation")
		}
		switch edge.Kind {
		case "continues", "branches", "joins", "fulfills", "reports":
		default:
			return Graph{}, fmt.Errorf("unsupported relation %s", edge.Kind)
		}
		for _, w := range edge.Witnesses {
			if _, ok := byObservation[w]; !ok {
				return Graph{}, fmt.Errorf("unknown relation witness %s", w)
			}
		}
	}
	// Only explicit activity membership creates strands. A shared actor, source
	// paragraph, place, or a shared graph node never unions activity identities.
	threadIndex := map[string]int{}
	for _, node := range g.Nodes {
		for _, change := range node.Changes {
			index, exists := threadIndex[change.ActivityID]
			if !exists {
				if change.Operation != "initiate" {
					return Graph{}, fmt.Errorf("activity %s has no initiation in this projection", change.ActivityID)
				}
				index = len(g.Threads)
				threadIndex[change.ActivityID] = index
				g.Threads = append(g.Threads, Thread{ActivityID: change.ActivityID, Status: "outcome_not_observed", NodeIDs: []string{}})
			} else if change.Operation == "initiate" {
				return Graph{}, fmt.Errorf("activity %s initiated twice", change.ActivityID)
			}
			thread := &g.Threads[index]
			if thread.Status == "completed" {
				return Graph{}, fmt.Errorf("activity %s changes after completion; use a new occurrence", change.ActivityID)
			}
			if !includes(thread.NodeIDs, node.ID) {
				thread.NodeIDs = append(thread.NodeIDs, node.ID)
			}
			if change.Operation == "complete" {
				thread.Status = "completed"
			}
		}
	}
	return g, nil
}

func includes(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}
