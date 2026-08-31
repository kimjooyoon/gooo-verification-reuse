package protocol

import (
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func RenderEvaluator(ir SemanticIR) ([]byte, error) {
	var builder strings.Builder
	builder.WriteString("package generated\n\n")
	fmt.Fprintf(&builder, "const Protocol = %s\n", strconv.Quote(ir.Protocol))
	fmt.Fprintf(&builder, "const SourceDigest = %s\n", strconv.Quote(ir.SourceDigest))
	fmt.Fprintf(&builder, "const SemanticDigest = %s\n\n", strconv.Quote(ir.Digest))
	builder.WriteString("type Activity struct {\n")
	builder.WriteString("\tOrdinal int\n\tID string\n\tName string\n\tStage string\n\tStep string\n\tProofChoice string\n\tIndicatorClass string\n\tClosedReason string\n\tDependsOn []string\n")
	builder.WriteString("}\n\nvar Activities = []Activity{\n")
	for _, cell := range ir.Cells {
		fmt.Fprintf(&builder, "\t{Ordinal: %d, ID: %s, Name: %s, Stage: %s, Step: %s, ProofChoice: %s, IndicatorClass: %s, ClosedReason: %s, DependsOn: []string{",
			cell.Ordinal, strconv.Quote(cell.ID), strconv.Quote(cell.Activity), strconv.Quote(cell.Stage), strconv.Quote(cell.Step), strconv.Quote(cell.ProofChoice), strconv.Quote(cell.IndicatorClass), strconv.Quote(cell.ClosedReason))
		for index, dependency := range cell.DependsOn {
			if index > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(strconv.Quote(dependency))
		}
		builder.WriteString("}},\n")
	}
	builder.WriteString("}\n\nfunc ActivityCount() int { return len(Activities) }\n")
	return format.Source([]byte(builder.String()))
}

func WriteJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func WriteText(path, value string) error {
	return os.WriteFile(path, []byte(value), 0o644)
}

func RenderHumanReport(receipt Receipt) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "# Verification evidence reuse report\n\n")
	fmt.Fprintf(&builder, "decision: `%s`\n", receipt.Decision)
	fmt.Fprintf(&builder, "scenario: `%s`\n", receipt.Scenario)
	fmt.Fprintf(&builder, "claim: `%s` / `%s`\n", receipt.Claim.State, receipt.Claim.Reason)
	fmt.Fprintf(&builder, "denominator: total=%d CLOSED=%d UNKNOWN=%d REFUTED=%d; precedence=REFUTED>UNKNOWN>CLOSED\n", receipt.Summary.Total, receipt.Summary.Closed, receipt.Summary.Unknown, receipt.Summary.Refuted)
	fmt.Fprintf(&builder, "execution counts: executed=%d reused=%d skipped=%d not_observed=%d\n", receipt.ExecutionCounts.Executed, receipt.ExecutionCounts.Reused, receipt.ExecutionCounts.Skipped, receipt.ExecutionCounts.NotObserved)
	fmt.Fprintf(&builder, "reuse plan: status=%s authorized=%t cache_hit=%t plan_only=%t actual_reused=%d consumer_test_executions=%d\n", receipt.Reuse.PlanStatus, receipt.Reuse.Authorized, receipt.Reuse.CacheHit, receipt.Reuse.PlanOnly, receipt.Reuse.ActualReused, receipt.Reuse.ConsumerTestExecutions)
	fmt.Fprintf(&builder, "authority: repository_writes=%d local_test_executions=%d cross_project_required_gates=%d output=%s verification=%s\n", receipt.Authority.RepositoryWrites, receipt.Authority.LocalTestExecutions, receipt.Authority.CrossProjectRequiredGates, receipt.Authority.OutputLocation, receipt.Authority.VerificationAuthority)
	fmt.Fprintf(&builder, "inventory: tree_files=%d tree_bytes=%d Go_files=%d Go_lines=%d Gooo_files=%d Gooo_lines=%d root_README_excluded=%t\n\n", receipt.Inventory.TreeFileCount, receipt.Inventory.TreeBytes, receipt.Inventory.GoFiles, receipt.Inventory.GoLines, receipt.Inventory.GoooFiles, receipt.Inventory.GoooLines, receipt.Inventory.RootReadmeExcluded)
	builder.WriteString("## Per-operation observations\n\n")
	for _, operation := range receipt.Operations {
		wall := "null"
		if operation.WallMS != nil {
			wall = fmt.Sprintf("%d", *operation.WallMS)
		}
		rss := "null"
		if operation.PeakRSSKiB != nil {
			rss = fmt.Sprintf("%d", *operation.PeakRSSKiB)
		}
		fmt.Fprintf(&builder, "- `%s` stage=%s status=%s wall_ms=%s peak_rss_kib=%s clock_domain=%s\n", operation.OperationID, operation.Stage, operation.Status, wall, rss, operation.ClockDomain)
	}
	builder.WriteString("\n## Cell claims\n\n")
	for _, cell := range receipt.Cells {
		fmt.Fprintf(&builder, "- `%s`: %s / %s", cell.CellID, cell.State, cell.Reason)
		if cell.Unknown != nil {
			fmt.Fprintf(&builder, " / unknown(stage=%s,step=%s,reason=%s,unknown_class=%s,next_operation=%s,blocked_by=%s)", cell.Unknown.Stage, cell.Unknown.Step, cell.Unknown.Reason, cell.Unknown.UnknownClass, cell.Unknown.NextOperation, strings.Join(cell.Unknown.BlockedBy, ","))
		}
		builder.WriteString("\n")
	}
	builder.WriteString("\nNo cache hit is treated as proof. No unlike clock domains are combined, no time is inferred for skipped or not-observed work, and no generalized claim is emitted.\n")
	return builder.String()
}

type Manifest struct {
	Schema                 string          `json:"schema"`
	Protocol               string          `json:"protocol"`
	Scenario               string          `json:"scenario"`
	SourceDigest           string          `json:"source_digest"`
	TreeDigest             string          `json:"tree_digest"`
	SemanticDigest         string          `json:"semantic_digest"`
	GeneratedEvaluatorDigest string        `json:"generated_evaluator_digest"`
	RootReadmeExcluded     bool            `json:"root_readme_excluded"`
	Files                  []ManifestFile `json:"files"`
	RepositoryWrites       int             `json:"repository_writes"`
	LocalTestExecutions    int             `json:"local_test_executions"`
	CrossProjectRequiredGates int          `json:"cross_project_required_gates"`
}

type ManifestFile struct {
	Path      string `json:"path"`
	Digest    string `json:"digest"`
	SizeBytes int64  `json:"size_bytes"`
}

func BuildManifest(root string, receipt Receipt) (Manifest, error) {
	paths := []string{"semantic-ir.json", "generated/evaluator.go", "verification-receipt.json", "human-report.md"}
	files := make([]ManifestFile, 0, len(paths))
	for _, path := range paths {
		digest, data, err := DigestFile(filepath.Join(root, path))
		if err != nil {
			return Manifest{}, err
		}
		files = append(files, ManifestFile{Path: path, Digest: digest, SizeBytes: int64(len(data))})
	}
	return Manifest{
		Schema: "gooo/verification-reuse/manifest/v1", Protocol: receipt.Protocol, Scenario: receipt.Scenario,
		SourceDigest: receipt.Source.Digest, TreeDigest: receipt.Source.TreeDigest, SemanticDigest: receipt.SemanticIR.Digest,
		GeneratedEvaluatorDigest: receipt.GeneratedEvaluator.Digest, RootReadmeExcluded: receipt.Inventory.RootReadmeExcluded,
		Files: files, RepositoryWrites: receipt.Authority.RepositoryWrites, LocalTestExecutions: receipt.Authority.LocalTestExecutions,
		CrossProjectRequiredGates: receipt.Authority.CrossProjectRequiredGates,
	}, nil
}
