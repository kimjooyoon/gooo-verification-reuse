package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/kimjooyoon/gooo-verification-reuse/internal/protocol"
)

func main() {
	if len(os.Args) < 2 {
		fatalf("usage: gooo-verification-reuse run [flags]")
	}
	var err error
	switch os.Args[1] {
	case "run":
		err = run(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q", os.Args[1])
	}
	if err != nil {
		fatalf("%v", err)
	}
}

func run(args []string) error {
	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	sourcePath := flags.String("source", "", "path to the .gooo source")
	contractPath := flags.String("contract", "", "path to the fixed denominator contract")
	corpusPath := flags.String("corpus", "", "path to the immutable fixture corpus")
	scenarioID := flags.String("scenario", "", "fixture scenario identifier")
	outPath := flags.String("out", "", "empty caller-owned output directory")
	treeRoot := flags.String("tree-root", ".", "source tree root used for the tree digest")
	buildObservationPath := flags.String("build-observation", "", "CI build observation JSON")
	testObservationPath := flags.String("test-observation", "", "CI test observation JSON")
	subjectSHA := flags.String("subject-sha", "UNBOUND", "subject commit identity")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *sourcePath == "" || *contractPath == "" || *corpusPath == "" || *scenarioID == "" || *outPath == "" {
		return fmt.Errorf("source, contract, corpus, scenario, and out are required")
	}

	absoluteTree, err := filepath.Abs(*treeRoot)
	if err != nil {
		return err
	}
	absoluteOut, err := filepath.Abs(*outPath)
	if err != nil {
		return err
	}
	if err := requireOutside(absoluteTree, absoluteOut); err != nil {
		return err
	}
	if err := prepareEmptyDirectory(absoluteOut); err != nil {
		return err
	}

	spec, sourceDigest, err := protocol.ParseSource(*sourcePath)
	if err != nil {
		return err
	}
	contract, contractDigest, err := protocol.LoadContract(*contractPath)
	if err != nil {
		return err
	}
	if spec.Protocol != "gooo/verification-reuse/v1" || len(spec.Cells) != contract.TargetCells || !reflect.DeepEqual(spec.Cells, contract.Cells) {
		return fmt.Errorf("source and fixed denominator are not identical")
	}
	corpus, fixtureDigest, err := protocol.LoadCorpus(*corpusPath)
	if err != nil {
		return err
	}
	scenario, err := protocol.FindScenario(corpus, *scenarioID)
	if err != nil {
		return err
	}
	treeDigest, inventory, err := protocol.TreeDigest(absoluteTree)
	if err != nil {
		return err
	}
	dependencyDigest, err := protocol.DigestNamedFiles(absoluteTree, []string{"go.mod", "go.sum"})
	if err != nil {
		return err
	}
	ir, err := protocol.BuildIR(*sourcePath, spec, sourceDigest)
	if err != nil {
		return err
	}
	evaluator, err := protocol.RenderEvaluator(ir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(absoluteOut, "generated"), 0o755); err != nil {
		return err
	}
	if err := protocol.WriteJSON(filepath.Join(absoluteOut, "semantic-ir.json"), ir); err != nil {
		return err
	}
	evaluatorPath := filepath.Join(absoluteOut, "generated", "evaluator.go")
	if err := protocol.WriteText(evaluatorPath, string(evaluator)); err != nil {
		return err
	}
	generatedDigest, _, err := protocol.DigestFile(evaluatorPath)
	if err != nil {
		return err
	}

	var buildObservation, testObservation *protocol.CommandObservation
	if *buildObservationPath != "" {
		observation, err := protocol.ReadObservation(*buildObservationPath)
		if err != nil {
			return err
		}
		buildObservation = &observation
	}
	if *testObservationPath != "" {
		observation, err := protocol.ReadObservation(*testObservationPath)
		if err != nil {
			return err
		}
		testObservation = &observation
	}
	receipt, err := protocol.Evaluate(contract, ir, scenario, sourceDigest, treeDigest, generatedDigest, contractDigest, fixtureDigest, dependencyDigest, buildObservation, testObservation, inventory)
	if err != nil {
		return err
	}
	receipt.SubjectSHA = *subjectSHA
	if err := protocol.WriteJSON(filepath.Join(absoluteOut, "verification-receipt.json"), receipt); err != nil {
		return err
	}
	if err := protocol.WriteText(filepath.Join(absoluteOut, "human-report.md"), protocol.RenderHumanReport(receipt)); err != nil {
		return err
	}
	manifest, err := protocol.BuildManifest(absoluteOut, receipt)
	if err != nil {
		return err
	}
	return protocol.WriteJSON(filepath.Join(absoluteOut, "manifest.json"), manifest)
}

func requireOutside(root, output string) error {
	relative, err := filepath.Rel(root, output)
	if err != nil {
		return err
	}
	if relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))) {
		return fmt.Errorf("output directory must be outside the source tree")
	}
	return nil
}

func prepareEmptyDirectory(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("output directory must be empty")
	}
	return nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
