package domain

import (
	"testing"
	"time"
)

func TestAssessEvidenceLevels(t *testing.T) {
	now := time.Now().Add(-time.Minute)
	tests := []struct {
		name              string
		leaf, trunk, pest Condition
		want              RiskLevel
	}{{"低风险", ConditionMild, ConditionHealthy, ConditionMild, RiskLow}, {"中风险", ConditionModerate, ConditionHealthy, ConditionMild, RiskMedium}, {"高风险", ConditionModerate, ConditionModerate, ConditionModerate, RiskHigh}, {"紧急树干缺陷", ConditionHealthy, ConditionSevere, ConditionHealthy, RiskCritical}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := NewEvidence("tree_1", "batch_1", EvidenceInput{InspectedAt: now, PhotoDigest: "sha256:test", LeafCondition: tt.leaf, TrunkDefect: tt.trunk, PestSigns: tt.pest, Notes: "结构化观察完整", SubmittedBy: "巡检员"})
			if err != nil {
				t.Fatal(err)
			}
			got, err := AssessEvidence(e)
			if err != nil {
				t.Fatal(err)
			}
			if got.Level != tt.want {
				t.Fatalf("risk=%s want %s, score=%d", got.Level, tt.want, got.Score)
			}
		})
	}
}

func TestCertificateDigestDetectsMutation(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	tree, err := RegisterTree("batch_1", NewTree{Name: "测试树", RoadLocation: "测试路 1 号", Species: "香樟", DiameterCM: 20, Responsibility: "一组"}, now)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := NewEvidence(tree.ID, tree.BatchID, EvidenceInput{InspectedAt: now, PhotoDigest: "sha256:test", LeafCondition: ConditionModerate, TrunkDefect: ConditionModerate, PestSigns: ConditionModerate, Notes: "测试证据", SubmittedBy: "巡检员"})
	if err != nil {
		t.Fatal(err)
	}
	if err = tree.Transition(TreeEvidenceSubmitted); err != nil {
		t.Fatal(err)
	}
	if err = tree.Transition(TreeAwaitingRemediation); err != nil {
		t.Fatal(err)
	}
	risk, _ := AssessEvidence(evidence)
	task, err := AssignRemediation(tree.ID, risk, "养护员", now.Add(time.Hour), now)
	if err != nil {
		t.Fatal(err)
	}
	if err = task.Complete("完成加固", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err = tree.Transition(TreeAwaitingRecheck); err != nil {
		t.Fatal(err)
	}
	if err = task.ApplyRecheck(true); err != nil {
		t.Fatal(err)
	}
	if err = tree.Transition(TreeClosed); err != nil {
		t.Fatal(err)
	}
	input := RecheckInput{RecheckedAt: now.Add(2 * time.Hour), Metrics: map[string]string{"树冠稳定": "是", "树干安全": "是", "病虫受控": "是"}, Result: "通过", Inspector: "复验员"}
	cert, err := IssueCertificate(tree, evidence, task, input, now.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyCertificate(cert, evidence.Digest, task.ID) {
		t.Fatal("新签发凭据应通过校验")
	}
	cert.Result = "不通过"
	if VerifyCertificate(cert, evidence.Digest, task.ID) {
		t.Fatal("篡改后的凭据不应通过校验")
	}
}
