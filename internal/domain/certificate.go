package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

type RecheckInput struct {
	RecheckedAt time.Time         `json:"recheckedAt"`
	Metrics     map[string]string `json:"metrics"`
	Result      string            `json:"result"`
	Inspector   string            `json:"inspector"`
}

type CareCertificate struct {
	ID          string            `json:"id"`
	TreeID      string            `json:"treeID"`
	RecheckedAt time.Time         `json:"recheckedAt"`
	Metrics     map[string]string `json:"metrics"`
	Result      string            `json:"result"`
	Inspector   string            `json:"inspector"`
	Digest      string            `json:"digest"`
	IssuedAt    time.Time         `json:"issuedAt"`
}

func EvaluateRecheck(input RecheckInput) (bool, error) {
	input.Result, input.Inspector = NormalizeText(input.Result), NormalizeText(input.Inspector)
	var errs ValidationErrors
	errs = AddRequired(errs, "result", input.Result)
	errs = AddRequired(errs, "inspector", input.Inspector)
	if input.RecheckedAt.IsZero() {
		errs = append(errs, FieldError{Field: "recheckedAt", Message: "不能为空"})
	}
	for _, key := range []string{"树冠稳定", "树干安全", "病虫受控"} {
		if NormalizeText(input.Metrics[key]) == "" {
			errs = append(errs, FieldError{Field: "metrics." + key, Message: "不能为空"})
		}
	}
	if len(errs) > 0 {
		return false, errs
	}
	return input.Result == "通过", nil
}

func IssueCertificate(tree TreeAsset, evidence InspectionEvidence, task RemediationTask, input RecheckInput, issuedAt time.Time) (CareCertificate, error) {
	passed, err := EvaluateRecheck(input)
	if err != nil {
		return CareCertificate{}, err
	}
	if !passed || task.Status != TaskClosed || tree.CurrentStatus != TreeClosed {
		return CareCertificate{}, ErrTransition
	}
	c := CareCertificate{ID: NewID("cert"), TreeID: tree.ID, RecheckedAt: input.RecheckedAt.UTC(), Metrics: CloneMetrics(input.Metrics), Result: input.Result, Inspector: input.Inspector, IssuedAt: issuedAt.UTC()}
	c.Digest = CertificateDigest(c, evidence.Digest, task.ID)
	return c, nil
}

func CertificateDigest(c CareCertificate, evidenceDigest, taskID string) string {
	values := []string{c.TreeID, c.RecheckedAt.Format(time.RFC3339Nano), c.Metrics["树冠稳定"], c.Metrics["树干安全"], c.Metrics["病虫受控"], c.Result, c.Inspector, evidenceDigest, taskID, c.IssuedAt.Format(time.RFC3339Nano)}
	sum := sha256.Sum256([]byte(strings.Join(values, "\x1f")))
	return hex.EncodeToString(sum[:])
}

func VerifyCertificate(c CareCertificate, evidenceDigest, taskID string) bool {
	return c.Digest != "" && c.Digest == CertificateDigest(c, evidenceDigest, taskID)
}

// CloneMetrics returns a shallow copy of in so callers cannot mutate the original map.
func CloneMetrics(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
