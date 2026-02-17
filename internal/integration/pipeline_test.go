package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/harakeishi/gopose/internal/generator"
	"github.com/harakeishi/gopose/internal/parser"
	"github.com/harakeishi/gopose/internal/scanner"
	"github.com/harakeishi/gopose/internal/testutil"
	"github.com/harakeishi/gopose/pkg/types"
)

// emptyNetworkDetector is a mock network detector that returns no networks.
// Used to avoid nil pointer dereference when calling DetectConflicts with
// no real Docker environment.
type emptyNetworkDetector struct{}

func (d *emptyNetworkDetector) DetectNetworks(ctx context.Context) ([]scanner.NetworkInfo, error) {
	return []scanner.NetworkInfo{}, nil
}

// testdataDir returns the absolute path to the integration test data directory.
func testdataDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.Abs("../../testdata/integration")
	if err != nil {
		t.Fatalf("failed to resolve testdata path: %v", err)
	}
	return dir
}

// TestPipeline_ParseToDetect_MultiService verifies that the pipeline correctly
// detects port conflicts when system-used ports overlap with compose service ports.
// compose-multi-service.yml defines: web:8080, api:3000, db:5432, redis:6379.
// With used ports [8080, 3000], at least 2 system conflicts should be detected.
func TestPipeline_ParseToDetect_MultiService(t *testing.T) {
	ctx := context.Background()
	log := testutil.NewTestLogger()

	// Step 1: Parse
	p := parser.NewYamlComposeParser(log)
	composePath := filepath.Join(testdataDir(t), "compose-multi-service.yml")
	config, err := p.ParseComposeFile(ctx, composePath)
	if err != nil {
		t.Fatalf("ParseComposeFile failed: %v", err)
	}

	if len(config.Services) != 4 {
		t.Fatalf("expected 4 services, got %d", len(config.Services))
	}

	// Step 2: Detect conflicts with used ports [8080, 3000]
	mockDetector := &testutil.MockPortDetector{UsedPorts: []int{8080, 3000}}
	detector := scanner.NewUnifiedConflictDetectorImpl(mockDetector, nil, log)

	conflicts, err := detector.DetectPortConflicts(ctx, config)
	if err != nil {
		t.Fatalf("DetectPortConflicts failed: %v", err)
	}

	if len(conflicts) < 2 {
		t.Errorf("expected at least 2 port conflicts, got %d", len(conflicts))
	}

	// Verify all detected conflicts are system type
	for _, c := range conflicts {
		if c.Type != types.ConflictTypeSystem {
			t.Errorf("expected conflict type %q, got %q for service %s port %d",
				types.ConflictTypeSystem, c.Type, c.ServiceName, c.Port)
		}
	}

	// Verify that conflicting ports are 8080 and 3000
	conflictPorts := make(map[int]bool)
	for _, c := range conflicts {
		conflictPorts[c.Port] = true
	}
	if !conflictPorts[8080] {
		t.Error("expected conflict on port 8080, but not found")
	}
	if !conflictPorts[3000] {
		t.Error("expected conflict on port 3000, but not found")
	}
}

// TestPipeline_ParseToDetect_NoConflict verifies that no conflicts are detected
// when there are no system-used ports overlapping with compose service ports.
func TestPipeline_ParseToDetect_NoConflict(t *testing.T) {
	ctx := context.Background()
	log := testutil.NewTestLogger()

	// Step 1: Parse
	p := parser.NewYamlComposeParser(log)
	composePath := filepath.Join(testdataDir(t), "compose-multi-service.yml")
	config, err := p.ParseComposeFile(ctx, composePath)
	if err != nil {
		t.Fatalf("ParseComposeFile failed: %v", err)
	}

	// Step 2: Detect conflicts with no used ports
	mockDetector := &testutil.MockPortDetector{UsedPorts: []int{}}
	detector := scanner.NewUnifiedConflictDetectorImpl(mockDetector, nil, log)

	conflicts, err := detector.DetectPortConflicts(ctx, config)
	if err != nil {
		t.Fatalf("DetectPortConflicts failed: %v", err)
	}

	if len(conflicts) != 0 {
		t.Errorf("expected 0 port conflicts, got %d", len(conflicts))
		for _, c := range conflicts {
			t.Logf("  unexpected conflict: service=%s port=%d type=%s", c.ServiceName, c.Port, c.Type)
		}
	}
}

// TestPipeline_ParseToDetect_NoPorts verifies that no conflicts are detected
// when the compose file has services without any port mappings.
func TestPipeline_ParseToDetect_NoPorts(t *testing.T) {
	ctx := context.Background()
	log := testutil.NewTestLogger()

	// Step 1: Parse
	p := parser.NewYamlComposeParser(log)
	composePath := filepath.Join(testdataDir(t), "compose-no-ports.yml")
	config, err := p.ParseComposeFile(ctx, composePath)
	if err != nil {
		t.Fatalf("ParseComposeFile failed: %v", err)
	}

	if len(config.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(config.Services))
	}

	// Step 2: Detect conflicts - even with used ports, no services have port mappings
	mockDetector := &testutil.MockPortDetector{UsedPorts: []int{8080, 3000, 5432}}
	detector := scanner.NewUnifiedConflictDetectorImpl(mockDetector, nil, log)

	conflicts, err := detector.DetectPortConflicts(ctx, config)
	if err != nil {
		t.Fatalf("DetectPortConflicts failed: %v", err)
	}

	if len(conflicts) != 0 {
		t.Errorf("expected 0 port conflicts for services without ports, got %d", len(conflicts))
	}
}

// TestPipeline_FullResolve_WritesOverride runs the full pipeline end-to-end:
// parse -> detect -> resolve -> generate -> validate -> write override file.
// It verifies that the output file exists and is non-empty.
func TestPipeline_FullResolve_WritesOverride(t *testing.T) {
	ctx := context.Background()
	log := testutil.NewTestLogger()

	// Step 1: Parse compose file
	p := parser.NewYamlComposeParser(log)
	composePath := filepath.Join(testdataDir(t), "compose-multi-service.yml")
	config, err := p.ParseComposeFile(ctx, composePath)
	if err != nil {
		t.Fatalf("ParseComposeFile failed: %v", err)
	}

	// Step 2: Detect conflicts (port 8080 is used by system)
	mockPortDetector := &testutil.MockPortDetector{UsedPorts: []int{8080}}
	conflictDetector := scanner.NewUnifiedConflictDetectorImpl(mockPortDetector, &emptyNetworkDetector{}, log)

	conflictInfo, err := conflictDetector.DetectConflicts(ctx, config, "testproject")
	if err != nil {
		t.Fatalf("DetectConflicts failed: %v", err)
	}

	if !conflictInfo.HasPortConflicts() {
		t.Fatal("expected port conflicts but none detected")
	}

	// Step 3: Resolve conflicts using mock port allocator
	// MockPortAllocator starts from NextPort and increments
	mockAllocator := &testutil.MockPortAllocator{NextPort: 18080}
	unifiedGen := generator.NewUnifiedOverrideGeneratorImpl(mockAllocator, log)

	portConfig := types.PortConfig{
		Range: types.PortRange{Start: 10000, End: 60000},
	}
	err = unifiedGen.ResolveConflicts(ctx, conflictInfo, types.ResolutionStrategyAutoIncrement, portConfig)
	if err != nil {
		t.Fatalf("ResolveConflicts failed: %v", err)
	}

	// Verify resolution was applied
	resolved := false
	for _, pc := range conflictInfo.PortConflicts {
		if pc.Port == 8080 && pc.Resolution != nil {
			resolved = true
			if pc.Resolution.ResolvedPort != 18080 {
				t.Errorf("expected resolved port 18080, got %d", pc.Resolution.ResolvedPort)
			}
		}
	}
	if !resolved {
		t.Fatal("port 8080 conflict was not resolved")
	}

	// Step 4: Generate override from conflicts
	override, err := unifiedGen.GenerateFromConflicts(ctx, config, conflictInfo)
	if err != nil {
		t.Fatalf("GenerateFromConflicts failed: %v", err)
	}

	if len(override.Services) == 0 {
		t.Fatal("expected at least 1 service in override, got 0")
	}

	// Step 5: Validate override
	overrideGen := generator.NewOverrideGeneratorImpl(log)
	err = overrideGen.ValidateOverride(ctx, override)
	if err != nil {
		t.Fatalf("ValidateOverride failed: %v", err)
	}

	// Step 6: Write override file to temp directory
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "docker-compose.override.yml")

	err = overrideGen.WriteOverrideFile(ctx, override, outputPath)
	if err != nil {
		t.Fatalf("WriteOverrideFile failed: %v", err)
	}

	// Step 7: Verify file exists and is non-empty
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("output file does not exist: %v", err)
	}
	if info.Size() == 0 {
		t.Error("output file is empty")
	}

	// Read and log the file content for debugging
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	t.Logf("Generated override file (%d bytes):\n%s", len(content), string(content))
}
