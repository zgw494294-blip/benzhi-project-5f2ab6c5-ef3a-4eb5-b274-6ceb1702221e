package domain

import "time"

type TaskStatus string

const (
	TaskAssigned        TaskStatus = "assigned"
	TaskInProgress      TaskStatus = "in_progress"
	TaskAwaitingRecheck TaskStatus = "awaiting_recheck"
	TaskReworkRequired  TaskStatus = "rework_required"
	TaskClosed          TaskStatus = "closed"
)

type RemediationTask struct {
	ID             string     `json:"id"`
	TreeID         string     `json:"treeID"`
	RiskLevel      RiskLevel  `json:"riskLevel"`
	TriggerRule    string     `json:"triggerRule"`
	Recommended    string     `json:"recommendedAction"`
	Assignee       string     `json:"assignee"`
	DueDate        time.Time  `json:"dueDate"`
	Status         TaskStatus `json:"status"`
	CompletionNote string     `json:"completionNote"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
	Version        int64      `json:"version"`
}

func AssignRemediation(treeID string, risk RiskAssessment, assignee string, dueDate time.Time, now time.Time) (RemediationTask, error) {
	assignee = NormalizeText(assignee)
	var errs ValidationErrors
	errs = AddRequired(errs, "treeID", treeID)
	errs = AddRequired(errs, "assignee", assignee)
	if !risk.Level.RequiresTask() {
		errs = append(errs, FieldError{Field: "riskLevel", Message: "低风险树木无需派发修复"})
	}
	if dueDate.Before(now) {
		errs = append(errs, FieldError{Field: "dueDate", Message: "截止日期不能早于当前时间"})
	}
	if len(errs) > 0 {
		return RemediationTask{}, errs
	}
	return RemediationTask{ID: NewID("task"), TreeID: treeID, RiskLevel: risk.Level, TriggerRule: risk.Rule, Recommended: risk.Recommended, Assignee: assignee, DueDate: dueDate.UTC(), Status: TaskAssigned, Version: 1}, nil
}

func (t *RemediationTask) Complete(note string, completedAt time.Time) error {
	note = NormalizeText(note)
	if t.Status != TaskAssigned && t.Status != TaskInProgress && t.Status != TaskReworkRequired {
		return ErrTransition
	}
	if note == "" || completedAt.IsZero() {
		return ValidationErrors{{Field: "completionNote", Message: "修复说明和完成时间不能为空"}}
	}
	t.CompletionNote, t.Status, t.Version = note, TaskAwaitingRecheck, t.Version+1
	completedAt = completedAt.UTC()
	t.CompletedAt = &completedAt
	return nil
}

func (t *RemediationTask) ApplyRecheck(passed bool) error {
	if t.Status != TaskAwaitingRecheck {
		return ErrTransition
	}
	if passed {
		t.Status = TaskClosed
	} else {
		t.Status = TaskReworkRequired
	}
	t.Version++
	return nil
}
