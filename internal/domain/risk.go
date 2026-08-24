package domain

import "fmt"

type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

type RiskAssessment struct {
	Score       int       `json:"score"`
	Level       RiskLevel `json:"level"`
	Rule        string    `json:"rule"`
	Recommended string    `json:"recommendedAction"`
}

func AssessEvidence(e InspectionEvidence) (RiskAssessment, error) {
	if !e.Complete() {
		return RiskAssessment{}, fmt.Errorf("%w: 证据摘要不完整", ErrValidation)
	}
	score := int(e.LeafCondition)*2 + int(e.TrunkDefect)*4 + int(e.PestSigns)*3
	assessment := RiskAssessment{Score: score}
	switch {
	case e.TrunkDefect == ConditionSevere || score >= 22:
		assessment.Level, assessment.Rule, assessment.Recommended = RiskCritical, "严重树干缺陷或综合分达到 22", "立即设置隔离区并组织专业加固或移除评估"
	case score >= 14:
		assessment.Level, assessment.Rule, assessment.Recommended = RiskHigh, "综合健康风险分达到 14", "48 小时内完成修枝、病害处置与结构加固"
	case score >= 7:
		assessment.Level, assessment.Rule, assessment.Recommended = RiskMedium, "综合健康风险分达到 7", "7 日内完成针对性施药、修剪或土壤改良"
	default:
		assessment.Level, assessment.Rule, assessment.Recommended = RiskLow, "综合健康风险分低于 7", "纳入常规养护并持续观察"
	}
	return assessment, nil
}

func (r RiskLevel) RequiresTask() bool { return r != RiskLow }
