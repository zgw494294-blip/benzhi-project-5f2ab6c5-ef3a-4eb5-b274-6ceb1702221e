package application

import "citytree/internal/domain"

type BatchView struct {
	Batch        domain.InspectionBatch   `json:"batch"`
	Trees        []TreeView               `json:"trees"`
	Total        int                      `json:"total"`
	Closed       int                      `json:"closed"`
	RiskTrees    int                      `json:"riskTrees"`
	Certificates []domain.CareCertificate `json:"certificates"`
}

type TreeView struct {
	Tree        domain.TreeAsset           `json:"tree"`
	Evidence    *domain.InspectionEvidence `json:"evidence,omitempty"`
	Task        *domain.RemediationTask    `json:"task,omitempty"`
	Certificate *domain.CareCertificate    `json:"certificate,omitempty"`
}

type DashboardView struct {
	Batches      []BatchSummary `json:"batches"`
	TotalTrees   int            `json:"totalTrees"`
	OpenTasks    int            `json:"openTasks"`
	Certificates int            `json:"certificates"`
}

type BatchSummary struct {
	Batch      domain.InspectionBatch `json:"batch"`
	TreeCount  int                    `json:"treeCount"`
	Closed     int                    `json:"closed"`
	RiskCount  int                    `json:"riskCount"`
	Completion int                    `json:"completion"`
}
