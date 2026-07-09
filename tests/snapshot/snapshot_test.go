package snapshot

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harald-mue/xsd-parser-go/internal/pipeline"
)

var update = flag.Bool("update", false, "regenerate expected snapshot files")

// This package runs the full pipeline against every fixture under
// tests/fixtures/<feature>/schema.xsd and compares the rendered models.go
// against the committed expected file.
//
// To regenerate snapshots after an intentional generator change run:
//
//	go test ./tests/snapshot -update

func TestSnapshots(t *testing.T) {
	fixturesDir := fixturesDir(t)

	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		t.Fatalf("read fixtures dir: %v", err)
	}

	ran := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		feature := entry.Name()
		schema := filepath.Join(fixturesDir, feature, "schema.xsd")
		if _, err := os.Stat(schema); err != nil {
			continue
		}
		ran++
		t.Run(feature, func(t *testing.T) {
			runSnapshot(t, feature, schema)
		})
	}
	if ran == 0 {
		t.Fatalf("no fixtures found under %s", fixturesDir)
	}
}

func runSnapshot(t *testing.T, feature string, schema string) {
	t.Helper()

	opts := pipeline.GenerateOptions{Package: "model"}
	fixtureDir := filepath.Dir(schema)
	if _, err := os.Stat(filepath.Join(fixtureDir, "optimizer_serde")); err == nil {
		opts.OptimizerSerde = true
	}
	if _, err := os.Stat(filepath.Join(fixtureDir, "catalog.xml")); err == nil {
		opts.Catalogs = []string{filepath.Join(fixtureDir, "catalog.xml")}
	}
	if _, err := os.Stat(filepath.Join(fixtureDir, "bindings.xjb")); err == nil {
		opts.BindingFiles = []string{filepath.Join(fixtureDir, "bindings.xjb")}
	}

	files, err := pipeline.RenderWithGenerateOptions([]string{schema}, opts)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	got, ok := files["models.go"]
	if !ok {
		t.Fatalf("rendered output does not contain models.go: %v", files)
	}

	expectedDir := filepath.Join(filepath.Dir(schema), "expected")
	expectedPath := filepath.Join(expectedDir, "model.go.golden")

	if *update {
		if err := os.MkdirAll(expectedDir, 0o755); err != nil {
			t.Fatalf("create expected dir: %v", err)
		}
		if err := os.WriteFile(expectedPath, got, 0o644); err != nil {
			t.Fatalf("write expected file: %v", err)
		}
		return
	}

	want, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("missing snapshot %s (run `go test ./tests/snapshot -update`): %v", expectedPath, err)
	}
	if string(got) != string(want) {
		t.Fatalf("snapshot mismatch for %s\n--- want ---\n%s\n--- got ---\n%s", feature, want, got)
	}
}

func fixturesDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	// tests/snapshot -> tests/fixtures
	candidate := filepath.Join(filepath.Dir(dir), "fixtures")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	t.Fatalf("fixtures directory not found relative to %s", dir)
	return ""
}

func TestMain(m *testing.M) {
	flag.Parse()
	// -update is registered above; ensure it is parsed even when `go test ./...`
	// passes other flags by ignoring the "flag provided but not defined" case.
	_ = strings.TrimSpace
	os.Exit(m.Run())
}
