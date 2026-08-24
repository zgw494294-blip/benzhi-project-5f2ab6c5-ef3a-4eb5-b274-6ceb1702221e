package domain

import "time"

type BatchStatus string

const (
	BatchOpen       BatchStatus = "open"
	BatchInProgress BatchStatus = "in_progress"
	BatchCompleted  BatchStatus = "completed"
)

type InspectionBatch struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Area      string      `json:"area"`
	Status    BatchStatus `json:"status"`
	Version   int64       `json:"version"`
	CreatedAt time.Time   `json:"createdAt"`
}

func NewInspectionBatch(name, area string, now time.Time) (InspectionBatch, error) {
	name, area = NormalizeText(name), NormalizeText(area)
	var errs ValidationErrors
	errs = AddRequired(errs, "name", name)
	errs = AddRequired(errs, "area", area)
	if len(errs) > 0 {
		return InspectionBatch{}, errs
	}
	return InspectionBatch{ID: NewID("bat"), Name: name, Area: area, Status: BatchOpen, Version: 1, CreatedAt: now.UTC()}, nil
}

func (b *InspectionBatch) MarkStarted() {
	if b.Status == BatchOpen {
		b.Status = BatchInProgress
		b.Version++
	}
}

func (b *InspectionBatch) MarkCompleted() {
	if b.Status != BatchCompleted {
		b.Status = BatchCompleted
		b.Version++
	}
}
