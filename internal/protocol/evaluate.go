package protocol

import (
	"fmt"
	"strings"
)

func Evaluate(contract Contract, ir SemanticIR, scenario Scenario, sourceDigest, treeDigest, generatedDigest, contractDigest, fixtureDigest, dependencyDigest string, build, test *CommandObservation, inventory Inventory) (Receipt, error) {
	if len(contract.Cells) != 12 || ir.CellCount != 12 {
		return Receipt{}, fmt.Errorf("evaluation requires the fixed twelve-cell denominator")
	}
	current := BuildCurrentBindings(sourceDigest, treeDigest, generatedDigest, dependencyDigest, fixtureDigest, contractDigest, scenario.PriorTerminalResult)
	values := map[string]string{
		"$SOURCE_DIGEST": sourceDigest, "$TREE_DIGEST": treeDigest, "$GENERATED_ARTIFACT_DIGEST": generatedDigest,
		"$GO_TOOLCHAIN": current.GoToolchainIdentity, "$COMMAND_SEMANTICS": current.CommandSemantics,
		"$DEPENDENCY_INPUTS": dependencyDigest, "$FIXTURE_DIGEST": fixtureDigest, "$POLICY_DIGEST": contractDigest,
		"$PLATFORM_CLOCK": current.PlatformClockDomain, "$PRIOR_RESULT": scenario.PriorTerminalResult,
		"$CLOCK_DOMAIN": current.PlatformClockDomain,
	}
	if build != nil {
		values["$BUILD_RESULT"] = build.ResultDigest
	}
	if test != nil {
		values["$TEST_RESULT"] = test.ResultDigest
	}
	prior := ResolveBindingTokens(scenario.PriorBindings, values)
	operations := ApplyObservations(scenario.Operations, build, test, values)
	decisions := make([]CellDecision, 0, len(contract.Cells))

	for _, cell := range contract.Cells[:9] {
		decisions = append(decisions, evaluateBindingCell(cell, current, prior))
	}
	decisions = append(decisions, evaluateReuseCell(contract.Cells[9], decisions, scenario))
	counts := countOperations(operations)
	decisions = append(decisions, evaluateOperationCell(contract.Cells[10], operations))
	decisions = append(decisions, evaluateReportCell(contract.Cells[11], decisions[10]))

	summary := summarize(decisions)
	topDecision, topReason, topUnknown := aggregateClaim(decisions)
	planDecision := decisions[9]
	actualReused := counts.Reused
	receipt := Receipt{
		Schema: "gooo/verification-reuse/verification-receipt/v1", Protocol: ir.Protocol, Scenario: scenario.ID,
		Decision: topDecision, Claim: Claim{State: topDecision, Reason: topReason, Unknown: topUnknown},
		Source: SourceIdentity{Path: ir.SourcePath, Digest: sourceDigest, TreeDigest: treeDigest},
		SemanticIR: ArtifactIdentity{Path: "semantic-ir.json", Digest: ir.Digest},
		GeneratedEvaluator: ArtifactIdentity{Path: "generated/evaluator.go", Digest: generatedDigest},
		ContractDigest: contractDigest, FixtureCorpusDigest: fixtureDigest, CurrentBindings: current, PriorBindings: prior,
		CacheHit: scenario.CacheHit, Reuse: ReuseReport{
			PlanStatus: planDecision.State, PlanReason: planDecision.Reason, Authorized: planDecision.State == Closed,
			CacheHit: scenario.CacheHit, PlanOnly: scenario.PlanOnly, ActualReused: actualReused, ConsumerTestExecutions: 0,
		}, Operations: operations, ExecutionCounts: counts, Summary: summary, Cells: decisions, Inventory: inventory,
		Authority: Authority{RepositoryWrites: 0, LocalTestExecutions: 0, CrossProjectRequiredGates: 0, OutputLocation: "CALLER_OWNED_TEMP_ONLY", VerificationAuthority: "GITHUB_ACTIONS"},
		ExternalReleaseInputs: []string{}, DevelopmentProvenance: Provenance{AppendOnly: true, ResetDeleteRewrite: false, FailedAttempts: []string{}, OptionalExternalInputs: []string{}},
	}
	return receipt, nil
}

func evaluateBindingCell(cell Cell, current, prior BindingSet) CellDecision {
	result := baseCell(cell)
	checks := []struct {
		name    string
		current string
		prior   string
	}{
		{"source_digest", current.SourceDigest, prior.SourceDigest},
		{"tree_digest", current.TreeDigest, prior.TreeDigest},
	}
	if cell.ID != "SOURCE_TREE_BINDING" {
		checks = []struct {
			name    string
			current string
			prior   string
		}{bindingForCell(cell.ID, current, prior)}
	}
	for _, check := range checks {
		if check.current == "" || check.prior == "" {
			result.State = Unknown
			result.Reason = strings.ToUpper(check.name) + "_MISSING"
			result.Unknown = unknownForField(cell, result.Reason, check.name)
			result.BlockedBy = []string{check.name}
			return result
		}
		if check.current != check.prior {
			result.State = Refuted
			result.Reason = strings.ToUpper(check.name) + "_MISMATCH"
			result.BlockedBy = []string{}
			return result
		}
	}
	if cell.ID == "PRIOR_RESULT_BINDING" && current.PriorTerminalResult != "PASS" {
		result.State = Refuted
		result.Reason = "PRIOR_TERMINAL_RESULT_FAILED"
		result.BlockedBy = []string{}
		return result
	}
	result.State = Closed
	result.Reason = cell.ClosedReason
	return result
}

func bindingForCell(id string, current, prior BindingSet) struct {
	name    string
	current string
	prior   string
} {
	switch id {
	case "GENERATED_ARTIFACT_BINDING":
		return struct {
			name    string
			current string
			prior   string
		}{"generated_artifact_digest", current.GeneratedArtifactDigest, prior.GeneratedArtifactDigest}
	case "TOOLCHAIN_BINDING":
		return struct {
			name    string
			current string
			prior   string
		}{"go_toolchain_identity", current.GoToolchainIdentity, prior.GoToolchainIdentity}
	case "COMMAND_BINDING":
		return struct {
			name    string
			current string
			prior   string
		}{"command_semantics", current.CommandSemantics, prior.CommandSemantics}
	case "DEPENDENCY_BINDING":
		return struct {
			name    string
			current string
			prior   string
		}{"dependency_inputs", current.DependencyInputs, prior.DependencyInputs}
	case "FIXTURE_BINDING":
		return struct {
			name    string
			current string
			prior   string
		}{"fixture_corpus_identity", current.FixtureCorpusIdentity, prior.FixtureCorpusIdentity}
	case "POLICY_BINDING":
		return struct {
			name    string
			current string
			prior   string
		}{"policy_identity", current.PolicyIdentity, prior.PolicyIdentity}
	case "PLATFORM_CLOCK_BINDING":
		return struct {
			name    string
			current string
			prior   string
		}{"platform_clock_domain", current.PlatformClockDomain, prior.PlatformClockDomain}
	case "PRIOR_RESULT_BINDING":
		return struct {
			name    string
			current string
			prior   string
		}{"prior_terminal_result", current.PriorTerminalResult, prior.PriorTerminalResult}
	default:
		return struct {
			name    string
			current string
			prior   string
		}{"unknown_binding", "", ""}
	}
}

func evaluateReuseCell(cell Cell, decisions []CellDecision, scenario Scenario) CellDecision {
	result := baseCell(cell)
	refuted := make([]string, 0)
	unknown := make([]string, 0)
	for _, decision := range decisions {
		if decision.State == Refuted {
			refuted = append(refuted, decision.CellID)
		}
		if decision.State == Unknown {
			unknown = append(unknown, decision.CellID)
		}
	}
	if len(refuted) > 0 {
		result.State = Refuted
		result.Reason = "REUSE_NOT_AUTHORIZED_AFTER_REFUTATION"
		result.BlockedBy = refuted
		return result
	}
	if len(unknown) > 0 {
		result.State = Unknown
		result.Reason = "REUSE_PLAN_BLOCKED_BY_UNKNOWN_BINDING"
		result.Unknown = dependencyUnknown(cell, result.Reason, unknown)
		result.BlockedBy = unknown
		return result
	}
	if !scenario.CacheHit {
		result.State = Unknown
		result.Reason = "CACHE_MISS_NOT_OBSERVED"
		result.Unknown = unknownForField(cell, result.Reason, "cache_hit")
		result.BlockedBy = []string{"cache_hit"}
		return result
	}
	result.State = Closed
	result.Reason = cell.ClosedReason
	return result
}

func evaluateOperationCell(cell Cell, operations []Operation) CellDecision {
	result := baseCell(cell)
	if len(operations) != 3 {
		result.State = Unknown
		result.Reason = "OPERATION_OBSERVATIONS_INCOMPLETE"
		result.Unknown = unknownForField(cell, result.Reason, "operation_observations")
		result.BlockedBy = []string{"operation_observations"}
		return result
	}
	unknownReasons := make([]string, 0)
	unknownBlocked := make([]string, 0)
	for _, operation := range operations {
		switch operation.Status {
		case "EXECUTED":
			if operation.WallMS == nil || operation.PeakRSSKiB == nil {
				unknownReasons = append(unknownReasons, operation.OperationID+":metrics")
				unknownBlocked = append(unknownBlocked, operation.OperationID)
				continue
			}
			if *operation.WallMS < 0 {
				result.State = Refuted
				result.Reason = "NEGATIVE_EXECUTED_DURATION"
				result.BlockedBy = []string{}
				return result
			}
			if operation.ClockDomain == "" {
				unknownReasons = append(unknownReasons, operation.OperationID+":clock_domain")
				unknownBlocked = append(unknownBlocked, operation.OperationID+":clock_domain")
			}
		case "REUSED", "SKIPPED", "NOT_OBSERVED":
			if operation.WallMS != nil || operation.PeakRSSKiB != nil {
				result.State = Refuted
				result.Reason = "EXECUTION_STATUS_METRIC_POLICY_CONFLICT"
				result.BlockedBy = []string{}
				return result
			}
		default:
			result.State = Refuted
			result.Reason = "UNRECOGNIZED_OPERATION_STATUS"
			result.BlockedBy = []string{}
			return result
		}
	}
	if len(unknownReasons) > 0 {
		result.State = Unknown
		result.Reason = "EXECUTION_METRIC_OBSERVATION_MISSING"
		result.Unknown = dependencyUnknown(cell, result.Reason, unknownBlocked)
		result.BlockedBy = unknownBlocked
		return result
	}
	result.State = Closed
	result.Reason = cell.ClosedReason
	return result
}

func evaluateReportCell(cell Cell, operationDecision CellDecision) CellDecision {
	result := baseCell(cell)
	if operationDecision.State == Refuted {
		result.State = Refuted
		result.Reason = "REPORT_BLOCKED_BY_REFUTED_OPERATION_ACCOUNTING"
		result.BlockedBy = []string{operationDecision.CellID}
		return result
	}
	if operationDecision.State == Unknown {
		result.State = Unknown
		result.Reason = "REPORT_BLOCKED_BY_UNKNOWN_OPERATION_ACCOUNTING"
		result.Unknown = dependencyUnknown(cell, result.Reason, []string{operationDecision.CellID})
		result.BlockedBy = []string{operationDecision.CellID}
		return result
	}
	result.State = Closed
	result.Reason = cell.ClosedReason
	return result
}

func baseCell(cell Cell) CellDecision {
	return CellDecision{CellID: cell.ID, Activity: cell.Activity, State: Closed, Reason: "", BlockedBy: []string{}}
}

func unknownForField(cell Cell, reason, field string) *Unknown {
	return &Unknown{Stage: cell.Stage, Step: cell.Step, Reason: reason, UnknownClass: "DIRECT_MISSING", NextOperation: "PROVIDE_" + strings.ToUpper(field) + "_BINDING", BlockedBy: []string{field}}
}

func dependencyUnknown(cell Cell, reason string, blocked []string) *Unknown {
	return &Unknown{Stage: cell.Stage, Step: cell.Step, Reason: reason, UnknownClass: "DEPENDENCY_BLOCKED", NextOperation: "RESTORE_UNKNOWN_PREDECESSORS", BlockedBy: append([]string{}, blocked...)}
}

func countOperations(operations []Operation) ExecutionCounts {
	var counts ExecutionCounts
	for _, operation := range operations {
		switch operation.Status {
		case "EXECUTED":
			counts.Executed++
		case "REUSED":
			counts.Reused++
		case "SKIPPED":
			counts.Skipped++
		case "NOT_OBSERVED":
			counts.NotObserved++
		}
	}
	return counts
}

func summarize(decisions []CellDecision) Summary {
	var summary Summary
	summary.Total = len(decisions)
	for _, decision := range decisions {
		switch decision.State {
		case Closed:
			summary.Closed++
		case Unknown:
			summary.Unknown++
		case Refuted:
			summary.Refuted++
		}
	}
	return summary
}

func aggregateClaim(decisions []CellDecision) (Decision, string, *Unknown) {
	for _, decision := range decisions {
		if decision.State == Refuted {
			return Refuted, decision.Reason, nil
		}
	}
	for _, decision := range decisions {
		if decision.State == Unknown {
			return Unknown, decision.Reason, decision.Unknown
		}
	}
	return Closed, "ALL_PROTOCOL_CELLS_CLOSED", nil
}
