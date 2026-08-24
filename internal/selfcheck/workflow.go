package selfcheck

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"citytree/internal/application"
	"citytree/internal/domain"
)

type Summary struct {
	BatchID           string `json:"batchID"`
	TreeID            string `json:"treeID"`
	RiskLevel         string `json:"riskLevel"`
	ConflictProtected bool   `json:"conflictProtected"`
	IdempotencyReused bool   `json:"idempotencyReused"`
	CertificateDigest string `json:"certificateDigest"`
}

func Run(ctx context.Context, baseURL string) (Summary, error) {
	client := NewClient(baseURL)
	if err := client.WaitReady(ctx); err != nil {
		return Summary{}, err
	}
	now := time.Now().UTC().Truncate(time.Second)
	create := application.CreateBatchCommand{Name: "中心大道夏季专项巡检", Area: "中心城区东段", Operator: "自检巡检员"}
	create.Tree.Name = "中心大道 001 号香樟"
	create.Tree.RoadLocation = "中心大道与园林路交叉口东北侧"
	create.Tree.Species = "香樟"
	create.Tree.DiameterCM = 38.5
	create.Tree.ResponsibilityArea = "东段养护一组"
	created, status, err := request[application.CreateBatchResult](ctx, client, http.MethodPost, "/api/batches", "selfcheck-create-0001", create)
	if err != nil || status != http.StatusCreated {
		return Summary{}, stepError("创建批次", err, status)
	}
	replayed, status, err := request[application.CreateBatchResult](ctx, client, http.MethodPost, "/api/batches", "selfcheck-create-0001", create)
	if err != nil || replayed.Batch.ID != created.Batch.ID {
		return Summary{}, fmt.Errorf("幂等重试未复用创建结果")
	}
	evidenceCmd := application.SubmitEvidenceCommand{ExpectedVersion: created.Tree.Version, InspectedAt: now, PhotoDigest: "sha256:selfcheck-photo-f26b8b", LeafCondition: domain.ConditionModerate, TrunkDefect: domain.ConditionModerate, PestSigns: domain.ConditionModerate, Notes: "叶缘失绿，主干有纵向裂纹并发现蛀孔", SubmittedBy: "自检巡检员"}
	evidence, status, err := request[application.SubmitEvidenceResult](ctx, client, http.MethodPost, "/api/trees/"+created.Tree.ID+"/evidence", "selfcheck-evidence-01", evidenceCmd)
	if err != nil || status != http.StatusCreated {
		return Summary{}, stepError("提交证据", err, status)
	}
	assessCmd := application.AssessCommand{ExpectedVersion: evidence.Tree.Version, Assignee: "自检养护员", DueDate: now.Add(72 * time.Hour), Operator: "自检负责人"}
	assessed, status, err := request[application.AssessResult](ctx, client, http.MethodPost, "/api/trees/"+created.Tree.ID+"/assess", "selfcheck-assess-0001", assessCmd)
	if err != nil || assessed.Task == nil || assessed.Assessment.Level != domain.RiskHigh {
		return Summary{}, stepError("风险评估", err, status)
	}
	conflictCmd := application.CompleteRemediationCommand{ExpectedTreeVersion: 1, ExpectedTaskVersion: assessed.Task.Version, CompletionNote: "错误版本不应写入", CompletedAt: now, Operator: "自检养护员"}
	_, conflictStatus, _ := request[application.CompleteRemediationResult](ctx, client, http.MethodPost, "/api/trees/"+created.Tree.ID+"/remediation", "selfcheck-conflict-01", conflictCmd)
	if conflictStatus != http.StatusConflict {
		return Summary{}, fmt.Errorf("版本冲突保护失败，状态码 %d", conflictStatus)
	}
	completeCmd := application.CompleteRemediationCommand{ExpectedTreeVersion: assessed.Tree.Version, ExpectedTaskVersion: assessed.Task.Version, CompletionNote: "清理蛀孔、修剪病枝并完成主干支撑加固，现场照片已归档", CompletedAt: now.Add(time.Hour), Operator: "自检养护员"}
	completed, status, err := request[application.CompleteRemediationResult](ctx, client, http.MethodPost, "/api/trees/"+created.Tree.ID+"/remediation", "selfcheck-remedy-001", completeCmd)
	if err != nil || completed.Tree.CurrentStatus != domain.TreeAwaitingRecheck {
		return Summary{}, stepError("确认修复", err, status)
	}
	recheckCmd := application.RecheckCommand{ExpectedTreeVersion: completed.Tree.Version, ExpectedTaskVersion: completed.Task.Version, RecheckedAt: now.Add(25 * time.Hour), Metrics: map[string]string{"树冠稳定": "病枝已清除，受力均衡", "树干安全": "裂纹封闭且支撑牢固", "病虫受控": "未见新增蛀孔"}, Result: "通过", Inspector: "自检复验员"}
	rechecked, status, err := request[application.RecheckResult](ctx, client, http.MethodPost, "/api/trees/"+created.Tree.ID+"/recheck", "selfcheck-recheck-001", recheckCmd)
	if err != nil || rechecked.Certificate == nil || rechecked.Tree.CurrentStatus != domain.TreeClosed {
		return Summary{}, stepError("复验签证", err, status)
	}
	if !domain.VerifyCertificate(*rechecked.Certificate, evidence.Evidence.Digest, completed.Task.ID) {
		return Summary{}, fmt.Errorf("养护凭据摘要验证失败")
	}
	batch, status, err := request[application.BatchView](ctx, client, http.MethodGet, "/api/batches/"+created.Batch.ID, "", nil)
	if err != nil || status != http.StatusOK || batch.Closed != 1 || batch.Batch.Status != domain.BatchCompleted {
		return Summary{}, stepError("查询批次完成度", err, status)
	}
	return Summary{BatchID: created.Batch.ID, TreeID: created.Tree.ID, RiskLevel: string(assessed.Assessment.Level), ConflictProtected: true, IdempotencyReused: true, CertificateDigest: rechecked.Certificate.Digest}, nil
}

func stepError(step string, err error, status int) error {
	if err != nil {
		return fmt.Errorf("%s失败: %w", step, err)
	}
	return fmt.Errorf("%s返回意外状态码 %d", step, status)
}
