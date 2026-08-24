package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type Condition int

const (
	ConditionHealthy Condition = iota
	ConditionMild
	ConditionModerate
	ConditionSevere
)

func (c Condition) Valid() bool { return c >= ConditionHealthy && c <= ConditionSevere }

type InspectionEvidence struct {
	ID            string    `json:"id"`
	TreeID        string    `json:"treeID"`
	BatchID       string    `json:"batchID"`
	InspectedAt   time.Time `json:"inspectedAt"`
	PhotoDigest   string    `json:"photoDigest"`
	LeafCondition Condition `json:"leafCondition"`
	TrunkDefect   Condition `json:"trunkDefect"`
	PestSigns     Condition `json:"pestSigns"`
	Notes         string    `json:"notes"`
	SubmittedBy   string    `json:"submittedBy"`
	Digest        string    `json:"digest"`
	Version       int64     `json:"version"`
}

type EvidenceInput struct {
	InspectedAt                           time.Time
	PhotoDigest, Notes, SubmittedBy       string
	LeafCondition, TrunkDefect, PestSigns Condition
}

func NewEvidence(treeID, batchID string, input EvidenceInput) (InspectionEvidence, error) {
	input.PhotoDigest = NormalizeText(input.PhotoDigest)
	input.Notes = NormalizeText(input.Notes)
	input.SubmittedBy = NormalizeText(input.SubmittedBy)
	var errs ValidationErrors
	errs = AddRequired(errs, "treeID", treeID)
	errs = AddRequired(errs, "batchID", batchID)
	errs = AddRequired(errs, "photoDigest", input.PhotoDigest)
	errs = AddRequired(errs, "notes", input.Notes)
	errs = AddRequired(errs, "submittedBy", input.SubmittedBy)
	if input.InspectedAt.IsZero() || input.InspectedAt.After(time.Now().Add(5*time.Minute)) {
		errs = append(errs, FieldError{Field: "inspectedAt", Message: "必须是有效且不晚于当前时间的时间"})
	}
	for field, value := range map[string]Condition{"leafCondition": input.LeafCondition, "trunkDefect": input.TrunkDefect, "pestSigns": input.PestSigns} {
		if !value.Valid() {
			errs = append(errs, FieldError{Field: field, Message: "必须是 0 到 3 的整数"})
		}
	}
	if len(errs) > 0 {
		return InspectionEvidence{}, errs
	}
	e := InspectionEvidence{ID: NewID("ev"), TreeID: treeID, BatchID: batchID, InspectedAt: input.InspectedAt.UTC(), PhotoDigest: input.PhotoDigest, LeafCondition: input.LeafCondition, TrunkDefect: input.TrunkDefect, PestSigns: input.PestSigns, Notes: input.Notes, SubmittedBy: input.SubmittedBy, Version: 1}
	e.Digest = e.CalculateDigest()
	return e, nil
}

func (e InspectionEvidence) CalculateDigest() string {
	parts := []string{e.TreeID, e.BatchID, e.InspectedAt.UTC().Format(time.RFC3339Nano), e.PhotoDigest, fmt.Sprint(e.LeafCondition), fmt.Sprint(e.TrunkDefect), fmt.Sprint(e.PestSigns), e.Notes, e.SubmittedBy}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x1f")))
	return hex.EncodeToString(sum[:])
}

func (e InspectionEvidence) Complete() bool {
	return e.Digest != "" && e.Digest == e.CalculateDigest() && e.PhotoDigest != "" && e.SubmittedBy != ""
}
