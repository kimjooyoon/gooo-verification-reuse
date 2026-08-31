package protocol

import "testing"

func TestClaimPrecedence(t *testing.T) {
	decision, reason, unknown := aggregateClaim([]CellDecision{
		{CellID: "missing", State: UnknownDecision, Reason: "SOURCE_DIGEST_MISSING", Unknown: &Unknown{Stage: "BINDING"}},
		{CellID: "contradiction", State: Refuted, Reason: "POLICY_IDENTITY_MISMATCH"},
	})
	if decision != Refuted || reason != "POLICY_IDENTITY_MISMATCH" || unknown != nil {
		t.Fatalf("precedence was not REFUTED: decision=%s reason=%s unknown=%v", decision, reason, unknown)
	}
}

func TestMissingBindingHasCausalFrontier(t *testing.T) {
	decision := evaluateBindingCell(Cell{ID: "POLICY_BINDING", Stage: "BINDING", Step: "COMPARE_POLICY_IDENTITY"}, BindingSet{PolicyIdentity: "sha256:current"}, BindingSet{})
	if decision.State != UnknownDecision || decision.Unknown == nil {
		t.Fatalf("missing policy binding was not UNKNOWN")
	}
	if decision.Unknown.UnknownClass != "DIRECT_MISSING" || len(decision.Unknown.BlockedBy) != 1 || decision.Unknown.BlockedBy[0] != "policy_identity" {
		t.Fatalf("unexpected causal frontier: %#v", decision.Unknown)
	}
}

func TestGeneratedEvaluatorContainsAllActivities(t *testing.T) {
	ir := SemanticIR{Protocol: "gooo/verification-reuse/v1", SourceDigest: "sha256:source", Digest: "sha256:ir", Cells: []Cell{{Ordinal: 1, ID: "ONE", Activity: "One"}}, CellCount: 1}
	data, err := RenderEvaluator(ir)
	if err != nil {
		t.Fatalf("render evaluator: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("generated evaluator was empty")
	}
}
