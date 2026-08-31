package protocol

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const CommandSemantics = "gooo/verification-reuse/v1:run:ci-first:plan-only"

func ParseSource(path string) (SourceSpec, string, error) {
	digest, data, err := DigestFile(path)
	if err != nil {
		return SourceSpec{}, "", err
	}
	var spec SourceSpec
	for lineNumber, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "|")
		switch fields[0] {
		case "protocol":
			if len(fields) != 2 || fields[1] == "" {
				return SourceSpec{}, "", fmt.Errorf("source line %d: malformed protocol", lineNumber+1)
			}
			spec.Protocol = fields[1]
		case "cell":
			if len(fields) != 10 {
				return SourceSpec{}, "", fmt.Errorf("source line %d: cell requires 10 fields", lineNumber+1)
			}
			ordinal, err := strconv.Atoi(fields[1])
			if err != nil || ordinal < 1 {
				return SourceSpec{}, "", fmt.Errorf("source line %d: invalid ordinal", lineNumber+1)
			}
			cell := Cell{
				Ordinal: ordinal, ID: fields[2], Activity: fields[3], Stage: fields[4], Step: fields[5],
				ProofChoice: fields[6], IndicatorClass: fields[7], ClosedReason: fields[8],
				DependsOn: splitList(fields[9]),
			}
			if cell.ID == "" || cell.Activity == "" || cell.Stage == "" || cell.Step == "" || cell.ClosedReason == "" {
				return SourceSpec{}, "", fmt.Errorf("source line %d: cell identity is incomplete", lineNumber+1)
			}
			spec.Cells = append(spec.Cells, cell)
		default:
			return SourceSpec{}, "", fmt.Errorf("source line %d: unknown record %q", lineNumber+1, fields[0])
		}
	}
	if spec.Protocol == "" {
		return SourceSpec{}, "", fmt.Errorf("source has no protocol")
	}
	for index, cell := range spec.Cells {
		if cell.Ordinal != index+1 {
			return SourceSpec{}, "", fmt.Errorf("source ordinal %d is not %d", cell.Ordinal, index+1)
		}
	}
	return spec, digest, nil
}

func LoadContract(path string) (Contract, string, error) {
	digest, data, err := DigestFile(path)
	if err != nil {
		return Contract{}, "", err
	}
	var contract Contract
	if err := json.Unmarshal(data, &contract); err != nil {
		return Contract{}, "", err
	}
	if err := ValidateContract(contract); err != nil {
		return Contract{}, "", err
	}
	return contract, digest, nil
}

func LoadCorpus(path string) (FixtureCorpus, string, error) {
	digest, data, err := DigestFile(path)
	if err != nil {
		return FixtureCorpus{}, "", err
	}
	var corpus FixtureCorpus
	if err := json.Unmarshal(data, &corpus); err != nil {
		return FixtureCorpus{}, "", err
	}
	if corpus.Schema != "gooo/verification-reuse/fixture-corpus/v1" || len(corpus.Cases) == 0 {
		return FixtureCorpus{}, "", fmt.Errorf("invalid fixture corpus")
	}
	return corpus, digest, nil
}

func ValidateContract(contract Contract) error {
	if contract.Schema != "gooo/verification-reuse/denominator/v1" || contract.ID == "" {
		return fmt.Errorf("invalid contract identity")
	}
	if contract.TargetCells != 12 || len(contract.Cells) != contract.TargetCells {
		return fmt.Errorf("denominator must contain exactly 12 cells")
	}
	if strings.Join(contract.Precedence, ",") != "REFUTED,UNKNOWN,CLOSED" {
		return fmt.Errorf("invalid decision precedence")
	}
	if strings.Join(contract.UnknownFields, ",") != "stage,step,reason,unknown_class,next_operation,blocked_by" {
		return fmt.Errorf("invalid UNKNOWN field contract")
	}
	seenIDs := map[string]bool{}
	seenActivities := map[string]bool{}
	for index, cell := range contract.Cells {
		if cell.Ordinal != index+1 || cell.ID == "" || cell.Activity == "" || seenIDs[cell.ID] || seenActivities[cell.Activity] {
			return fmt.Errorf("contract cell identity is not unique and ordered")
		}
		seenIDs[cell.ID] = true
		seenActivities[cell.Activity] = true
	}
	return nil
}

func BuildIR(sourcePath string, spec SourceSpec, sourceDigest string) (SemanticIR, error) {
	payload := struct {
		Schema       string `json:"schema"`
		Protocol     string `json:"protocol"`
		SourcePath   string `json:"source_path"`
		SourceDigest string `json:"source_digest"`
		Cells        []Cell `json:"cells"`
		CellCount    int    `json:"cell_count"`
	}{
		Schema: "gooo/verification-reuse/semantic-ir/v1", Protocol: spec.Protocol,
		SourcePath: filepath.ToSlash(sourcePath), SourceDigest: sourceDigest,
		Cells: spec.Cells, CellCount: len(spec.Cells),
	}
	canonical, err := json.Marshal(payload)
	if err != nil {
		return SemanticIR{}, err
	}
	return SemanticIR{
		Schema: payload.Schema, Protocol: payload.Protocol, SourcePath: payload.SourcePath,
		SourceDigest: payload.SourceDigest, Cells: payload.Cells, CellCount: payload.CellCount,
		Digest: DigestBytes(canonical),
	}, nil
}

func FindScenario(corpus FixtureCorpus, id string) (Scenario, error) {
	for _, scenario := range corpus.Cases {
		if scenario.ID == id {
			return scenario, nil
		}
	}
	return Scenario{}, fmt.Errorf("scenario %q not found", id)
}

func BuildCurrentBindings(sourceDigest, treeDigest, generatedDigest, dependencyDigest, fixtureDigest, policyDigest, priorResult string) BindingSet {
	platform := runtime.Version() + "|" + runtime.GOOS + "|" + runtime.GOARCH + "|github.actions.monotonic.v1"
	return BindingSet{
		SourceDigest: sourceDigest, TreeDigest: treeDigest, GeneratedArtifactDigest: generatedDigest,
		GoToolchainIdentity: platform, CommandSemantics: CommandSemantics, DependencyInputs: dependencyDigest,
		FixtureCorpusIdentity: fixtureDigest, PolicyIdentity: policyDigest,
		PlatformClockDomain: runtime.GOOS + "/" + runtime.GOARCH + "/github.actions.monotonic.v1",
		PriorTerminalResult: priorResult,
	}
}

func ResolveBindingTokens(input BindingSet, values map[string]string) BindingSet {
	resolve := func(value string) string {
		if strings.HasPrefix(value, "$") {
			return values[value]
		}
		return value
	}
	return BindingSet{
		SourceDigest: resolve(input.SourceDigest), TreeDigest: resolve(input.TreeDigest),
		GeneratedArtifactDigest: resolve(input.GeneratedArtifactDigest), GoToolchainIdentity: resolve(input.GoToolchainIdentity),
		CommandSemantics: resolve(input.CommandSemantics), DependencyInputs: resolve(input.DependencyInputs),
		FixtureCorpusIdentity: resolve(input.FixtureCorpusIdentity), PolicyIdentity: resolve(input.PolicyIdentity),
		PlatformClockDomain: resolve(input.PlatformClockDomain), PriorTerminalResult: resolve(input.PriorTerminalResult),
	}
}

func ReadObservation(path string) (CommandObservation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CommandObservation{}, err
	}
	var observation CommandObservation
	if err := json.Unmarshal(data, &observation); err != nil {
		return CommandObservation{}, err
	}
	return observation, nil
}

func ApplyObservations(operations []Operation, build, test *CommandObservation, values map[string]string) []Operation {
	result := make([]Operation, len(operations))
	copy(result, operations)
	for index := range result {
		operation := &result[index]
		operation.ClockDomain = resolveValue(operation.ClockDomain, values)
		operation.ResultDigest = resolveValue(operation.ResultDigest, values)
		var observation *CommandObservation
		switch operation.Stage {
		case "BUILD":
			observation = build
		case "TEST":
			observation = test
		}
		if observation != nil && operation.WallMS == nil && operation.PeakRSSKiB == nil {
			wall := observation.WallMS
			rss := observation.PeakRSSKiB
			operation.WallMS = &wall
			operation.PeakRSSKiB = &rss
			operation.OperationID = observation.OperationID
			operation.ClockDomain = observation.ClockDomain
			operation.ResultDigest = observation.ResultDigest
			operation.Status = observation.Status
		}
	}
	return result
}

func resolveValue(value string, values map[string]string) string {
	if strings.HasPrefix(value, "$") {
		return values[value]
	}
	return value
}

func splitList(value string) []string {
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			result = append(result, strings.TrimSpace(part))
		}
	}
	return result
}
