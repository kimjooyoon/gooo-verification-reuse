package protocol

import "encoding/json"

type Decision string

const (
	Closed  Decision = "CLOSED"
	Unknown Decision = "UNKNOWN"
	Refuted Decision = "REFUTED"
)

type Unknown struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type Claim struct {
	State   Decision `json:"state"`
	Reason  string   `json:"reason"`
	Unknown *Unknown `json:"unknown,omitempty"`
}

type Cell struct {
	Ordinal        int      `json:"ordinal"`
	ID             string   `json:"id"`
	Activity       string   `json:"activity"`
	Stage          string   `json:"stage"`
	Step           string   `json:"step"`
	ProofChoice    string   `json:"proof_choice"`
	IndicatorClass string   `json:"indicator_class"`
	ClosedReason   string   `json:"closed_reason"`
	DependsOn      []string `json:"depends_on"`
}

type Contract struct {
	Schema          string   `json:"schema"`
	ID              string   `json:"id"`
	TargetCells     int      `json:"target_cells"`
	Precedence      []string `json:"precedence"`
	UnknownFields   []string `json:"unknown_fields"`
	ProofTotals     []Total  `json:"proof_totals"`
	IndicatorTotals []Total  `json:"indicator_totals"`
	Cells           []Cell   `json:"cells"`
}

type Total struct {
	ProofChoice    string `json:"proof_choice,omitempty"`
	IndicatorClass string `json:"indicator_class,omitempty"`
	Total          int    `json:"total"`
}

type SourceSpec struct {
	Protocol string `json:"protocol"`
	Cells    []Cell `json:"cells"`
}

type SemanticIR struct {
	Schema       string `json:"schema"`
	Protocol     string `json:"protocol"`
	SourcePath   string `json:"source_path"`
	SourceDigest string `json:"source_digest"`
	Cells        []Cell `json:"cells"`
	CellCount    int    `json:"cell_count"`
	Digest       string `json:"digest"`
}

type BindingSet struct {
	SourceDigest            string `json:"source_digest"`
	TreeDigest              string `json:"tree_digest"`
	GeneratedArtifactDigest string `json:"generated_artifact_digest"`
	GoToolchainIdentity     string `json:"go_toolchain_identity"`
	CommandSemantics        string `json:"command_semantics"`
	DependencyInputs        string `json:"dependency_inputs"`
	FixtureCorpusIdentity   string `json:"fixture_corpus_identity"`
	PolicyIdentity          string `json:"policy_identity"`
	PlatformClockDomain     string `json:"platform_clock_domain"`
	PriorTerminalResult     string `json:"prior_terminal_result"`
}

type FixtureCorpus struct {
	Schema   string     `json:"schema"`
	CorpusID string     `json:"corpus_id"`
	Cases    []Scenario `json:"cases"`
}

type Scenario struct {
	ID                  string      `json:"id"`
	Description         string      `json:"description"`
	CacheHit            bool        `json:"cache_hit"`
	PlanOnly            bool        `json:"plan_only"`
	PriorTerminalResult string      `json:"prior_terminal_result"`
	PriorBindings       BindingSet  `json:"prior_bindings"`
	Operations          []Operation `json:"operations"`
}

type Operation struct {
	OperationID  string `json:"operation_id"`
	Stage        string `json:"stage"`
	Status       string `json:"status"`
	WallMS       *int64 `json:"wall_ms"`
	PeakRSSKiB   *int64 `json:"peak_rss_kib"`
	ClockDomain  string `json:"clock_domain"`
	ResultDigest string `json:"result_digest"`
	Reason       string `json:"reason,omitempty"`
}

type CommandObservation struct {
	Status         string `json:"status"`
	OperationID    string `json:"operation_id"`
	WallMS         int64  `json:"wall_ms"`
	PeakRSSKiB     int64  `json:"peak_rss_kib"`
	ClockDomain    string `json:"clock_domain"`
	ResultDigest   string `json:"result_digest"`
	TerminalResult string `json:"terminal_result"`
}

type CellDecision struct {
	CellID    string   `json:"cell_id"`
	Activity  string   `json:"activity"`
	State     Decision `json:"state"`
	Reason    string   `json:"reason"`
	Unknown   *Unknown `json:"unknown,omitempty"`
	BlockedBy []string `json:"blocked_by"`
}

type ExecutionCounts struct {
	Executed    int `json:"executed"`
	Reused      int `json:"reused"`
	Skipped     int `json:"skipped"`
	NotObserved int `json:"not_observed"`
}

type Summary struct {
	Total   int `json:"total"`
	Closed  int `json:"closed"`
	Unknown int `json:"unknown"`
	Refuted int `json:"refuted"`
}

type Inventory struct {
	TreeFileCount      int64 `json:"tree_file_count"`
	TreeBytes          int64 `json:"tree_bytes"`
	GoFiles            int64 `json:"go_files"`
	GoLines            int64 `json:"go_lines"`
	GoooFiles          int64 `json:"gooo_files"`
	GoooLines          int64 `json:"gooo_lines"`
	RootReadmeExcluded bool  `json:"root_readme_excluded"`
}

type ReuseReport struct {
	PlanStatus             Decision `json:"plan_status"`
	PlanReason             string   `json:"plan_reason"`
	Authorized             bool     `json:"authorized"`
	CacheHit               bool     `json:"cache_hit"`
	PlanOnly               bool     `json:"plan_only"`
	ActualReused           int      `json:"actual_reused"`
	ConsumerTestExecutions int      `json:"consumer_test_executions"`
}

type Authority struct {
	RepositoryWrites          int    `json:"repository_writes"`
	LocalTestExecutions       int    `json:"local_test_executions"`
	CrossProjectRequiredGates int    `json:"cross_project_required_gates"`
	OutputLocation            string `json:"output_location"`
	VerificationAuthority     string `json:"verification_authority"`
}

type Provenance struct {
	AppendOnly             bool     `json:"append_only"`
	ResetDeleteRewrite     bool     `json:"reset_delete_rewrite"`
	FailedAttempts         []string `json:"failed_attempts"`
	OptionalExternalInputs []string `json:"optional_external_inputs"`
	DirectMainCommitCount  int      `json:"direct_main_commit_count"`
	DirectMainCommitSHA    string   `json:"direct_main_commit_sha"`
	DirectMainCommitPaths  []string `json:"direct_main_commit_paths"`
}

type Receipt struct {
	SubjectSHA            string           `json:"subject_sha"`
	Schema                string           `json:"schema"`
	Protocol              string           `json:"protocol"`
	Scenario              string           `json:"scenario"`
	Decision              Decision         `json:"decision"`
	Claim                 Claim            `json:"claim"`
	Source                SourceIdentity   `json:"source"`
	SemanticIR            ArtifactIdentity `json:"semantic_ir"`
	GeneratedEvaluator    ArtifactIdentity `json:"generated_evaluator"`
	ContractDigest        string           `json:"contract_digest"`
	FixtureCorpusDigest   string           `json:"fixture_corpus_digest"`
	CurrentBindings       BindingSet       `json:"current_bindings"`
	PriorBindings         BindingSet       `json:"prior_bindings"`
	CacheHit              bool             `json:"cache_hit"`
	Reuse                 ReuseReport      `json:"reuse"`
	Operations            []Operation      `json:"operations"`
	ExecutionCounts       ExecutionCounts  `json:"execution_counts"`
	Summary               Summary          `json:"summary"`
	Cells                 []CellDecision   `json:"cells"`
	Inventory             Inventory        `json:"inventory"`
	Authority             Authority        `json:"authority"`
	ExternalReleaseInputs []string         `json:"external_release_inputs"`
	DevelopmentProvenance Provenance       `json:"development_provenance"`
}

type SourceIdentity struct {
	Path       string `json:"path"`
	Digest     string `json:"digest"`
	TreeDigest string `json:"tree_digest"`
}

type ArtifactIdentity struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

func (b BindingSet) MarshalJSON() ([]byte, error) {
	type alias BindingSet
	return json.Marshal(alias(b))
}
