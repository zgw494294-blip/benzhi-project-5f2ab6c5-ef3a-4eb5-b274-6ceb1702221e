package domain

import "time"

type TreeStatus string

const (
	TreeRegistered          TreeStatus = "registered"
	TreeEvidenceSubmitted   TreeStatus = "evidence_submitted"
	TreeAwaitingRemediation TreeStatus = "awaiting_remediation"
	TreeAwaitingRecheck     TreeStatus = "awaiting_recheck"
	TreeClosed              TreeStatus = "closed"
)

type TreeAsset struct {
	ID             string     `json:"id"`
	BatchID        string     `json:"batchID"`
	Name           string     `json:"name"`
	RoadLocation   string     `json:"roadLocation"`
	Species        string     `json:"species"`
	DiameterCM     float64    `json:"diameterCM"`
	Responsibility string     `json:"responsibilityArea"`
	CurrentStatus  TreeStatus `json:"currentStatus"`
	Version        int64      `json:"version"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type NewTree struct {
	Name, RoadLocation, Species, Responsibility string
	DiameterCM                                  float64
}

func RegisterTree(batchID string, input NewTree, now time.Time) (TreeAsset, error) {
	input.Name = NormalizeText(input.Name)
	input.RoadLocation = NormalizeText(input.RoadLocation)
	input.Species = NormalizeText(input.Species)
	input.Responsibility = NormalizeText(input.Responsibility)
	var errs ValidationErrors
	errs = AddRequired(errs, "batchID", batchID)
	errs = AddRequired(errs, "name", input.Name)
	errs = AddRequired(errs, "roadLocation", input.RoadLocation)
	errs = AddRequired(errs, "species", input.Species)
	errs = AddRequired(errs, "responsibilityArea", input.Responsibility)
	if input.DiameterCM <= 0 || input.DiameterCM > 500 {
		errs = append(errs, FieldError{Field: "diameterCM", Message: "必须在 0 到 500 之间"})
	}
	if len(errs) > 0 {
		return TreeAsset{}, errs
	}
	return TreeAsset{ID: NewID("tree"), BatchID: batchID, Name: input.Name, RoadLocation: input.RoadLocation, Species: input.Species, DiameterCM: input.DiameterCM, Responsibility: input.Responsibility, CurrentStatus: TreeRegistered, Version: 1, CreatedAt: now.UTC()}, nil
}

func (t *TreeAsset) Transition(next TreeStatus) error {
	allowed := map[TreeStatus][]TreeStatus{
		TreeRegistered:          {TreeEvidenceSubmitted},
		TreeEvidenceSubmitted:   {TreeAwaitingRemediation, TreeClosed},
		TreeAwaitingRemediation: {TreeAwaitingRecheck},
		TreeAwaitingRecheck:     {TreeClosed, TreeAwaitingRemediation},
	}
	for _, candidate := range allowed[t.CurrentStatus] {
		if candidate == next {
			t.CurrentStatus = next
			t.Version++
			return nil
		}
	}
	return ErrTransition
}
