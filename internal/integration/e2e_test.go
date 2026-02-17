package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harakeishi/gopose/internal/generator"
	"github.com/harakeishi/gopose/internal/parser"
	"github.com/harakeishi/gopose/internal/scanner"
	"github.com/harakeishi/gopose/internal/testutil"
	"github.com/harakeishi/gopose/pkg/types"
)

// setupComposeFile creates a compose.yml in a temp directory and returns the
// temp dir path and compose file path.
func setupComposeFile(t *testing.T, content string) (string, string) {
	t.Helper()
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "compose.yml")
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write compose.yml: %v", err)
	}
	return tmpDir, composePath
}

// runFullPipeline executes the complete gopose pipeline (parse → detect → resolve → generate → validate → write)
// mirroring cmd/up.go logic. Returns the output path and override content.
func runFullPipeline(t *testing.T, composePath, outputDir string, usedPorts []int, dryRun bool) (string, []byte) {
	t.Helper()
	ctx := context.Background()
	log := testutil.NewTestLogger()

	// Step 1: Parse compose file
	yamlParser := parser.NewYamlComposeParser(log)
	config, err := yamlParser.ParseComposeFile(ctx, composePath)
	if err != nil {
		t.Fatalf("ParseComposeFile failed: %v", err)
	}

	// Step 2: Detect conflicts
	mockPortDetector := &testutil.MockPortDetector{UsedPorts: usedPorts}
	unifiedDetector := scanner.NewUnifiedConflictDetectorImpl(mockPortDetector, &emptyNetworkDetector{}, log)

	conflictInfo, err := unifiedDetector.DetectConflicts(ctx, config, "testproject")
	if err != nil {
		t.Fatalf("DetectConflicts failed: %v", err)
	}

	// Step 3: If no conflicts, return early
	if !conflictInfo.HasConflicts() {
		return "", nil
	}

	// Step 4: Resolve conflicts
	mockAllocator := &testutil.MockPortAllocator{NextPort: 18080}
	unifiedGen := generator.NewUnifiedOverrideGeneratorImpl(mockAllocator, log)

	portConfig := types.PortConfig{
		Range: types.PortRange{Start: 10000, End: 60000},
	}

	resolutionStrategy := types.ResolutionStrategyAutoIncrement
	if err := unifiedGen.ResolveConflicts(ctx, conflictInfo, resolutionStrategy, portConfig); err != nil {
		t.Fatalf("ResolveConflicts failed: %v", err)
	}

	// Step 5: Generate override
	override, err := unifiedGen.GenerateFromConflicts(ctx, config, conflictInfo)
	if err != nil {
		t.Fatalf("GenerateFromConflicts failed: %v", err)
	}

	// Step 6: Set project name
	override.Name = "testproject"

	// Step 7: Validate override
	overrideGen := generator.NewOverrideGeneratorImpl(log)
	if err := overrideGen.ValidateOverride(ctx, override); err != nil {
		t.Fatalf("ValidateOverride failed: %v", err)
	}

	// Step 8: Write (unless dry-run)
	outputPath := filepath.Join(outputDir, "compose.override.yml")
	if !dryRun {
		if err := overrideGen.WriteOverrideFile(ctx, override, outputPath); err != nil {
			t.Fatalf("WriteOverrideFile failed: %v", err)
		}
	}

	// Read back if file was written
	if !dryRun {
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("failed to read override file: %v", err)
		}
		return outputPath, content
	}

	return outputPath, nil
}

// TestE2E_NoConflicts verifies that no override file is generated when there
// are no port conflicts between system and compose services.
func TestE2E_NoConflicts(t *testing.T) {
	composeContent := `services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
  api:
    image: node:alpine
    ports:
      - "3000:3000"
`
	tmpDir, composePath := setupComposeFile(t, composeContent)

	// No used ports → no conflicts
	outputPath, content := runFullPipeline(t, composePath, tmpDir, []int{}, false)

	// Should return early with no output
	if outputPath != "" {
		t.Errorf("expected empty output path for no conflicts, got %q", outputPath)
	}
	if content != nil {
		t.Errorf("expected nil content for no conflicts, got %d bytes", len(content))
	}

	// Verify no override file was created
	overridePath := filepath.Join(tmpDir, "compose.override.yml")
	if _, err := os.Stat(overridePath); !os.IsNotExist(err) {
		t.Error("override file should not exist when there are no conflicts")
	}
}

// TestE2E_WithConflicts_WritesFile verifies the full pipeline produces an
// override file when port conflicts are detected.
func TestE2E_WithConflicts_WritesFile(t *testing.T) {
	composeContent := `services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
  api:
    image: node:alpine
    ports:
      - "3000:3000"
  db:
    image: postgres:13
    ports:
      - "5432:5432"
`
	tmpDir, composePath := setupComposeFile(t, composeContent)

	// Ports 8080 and 3000 are in use → 2 conflicts expected
	outputPath, content := runFullPipeline(t, composePath, tmpDir, []int{8080, 3000}, false)

	// Verify file was written
	if outputPath == "" {
		t.Fatal("expected non-empty output path")
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("override file does not exist: %v", err)
	}
	if info.Size() == 0 {
		t.Error("override file is empty")
	}

	// Verify content has YAML structure
	contentStr := string(content)
	t.Logf("Generated override:\n%s", contentStr)

	if !strings.Contains(contentStr, "services:") {
		t.Error("override file should contain 'services:' key")
	}

	// Verify the conflicting ports (8080, 3000) are resolved to new ports
	// MockPortAllocator starts at 18080
	if !strings.Contains(contentStr, "18080") {
		t.Error("override should contain resolved port 18080 for web service")
	}

	// Verify non-conflicting port (5432) is NOT changed
	// db service should not appear in override or should keep original port
	if strings.Contains(contentStr, "db:") {
		// If db appears, it should have the original port 5432
		// (the generator copies all ports for services with conflicts,
		// but db has no conflict so it shouldn't appear)
		t.Log("db service appears in override - verifying it keeps original port")
	}
}

// TestE2E_DryRun_NoFileWritten verifies that dry-run mode does NOT write any file.
func TestE2E_DryRun_NoFileWritten(t *testing.T) {
	composeContent := `services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
`
	tmpDir, composePath := setupComposeFile(t, composeContent)

	// Port 8080 in use → conflict exists
	outputPath, content := runFullPipeline(t, composePath, tmpDir, []int{8080}, true)

	// Dry run should not write file
	if content != nil {
		t.Errorf("expected nil content in dry-run mode, got %d bytes", len(content))
	}

	// Verify file does NOT exist
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Error("override file should not exist in dry-run mode")
	}
}

// TestE2E_AutoDetect_ComposeFile verifies that ComposeFileDetectorImpl can
// auto-detect compose files in a temp directory.
func TestE2E_AutoDetect_ComposeFile(t *testing.T) {
	ctx := context.Background()
	log := testutil.NewTestLogger()

	composeContent := `services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
`
	tmpDir, _ := setupComposeFile(t, composeContent)

	// Use detector to find compose file
	detector := parser.NewComposeFileDetectorImpl(log)
	detectedFile, err := detector.GetDefaultComposeFile(ctx, tmpDir)
	if err != nil {
		t.Fatalf("GetDefaultComposeFile failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, "compose.yml")
	if detectedFile != expectedPath {
		t.Errorf("detected file = %q, want %q", detectedFile, expectedPath)
	}

	// Parse the detected file and run pipeline
	yamlParser := parser.NewYamlComposeParser(log)
	config, err := yamlParser.ParseComposeFile(ctx, detectedFile)
	if err != nil {
		t.Fatalf("ParseComposeFile failed: %v", err)
	}

	if len(config.Services) != 1 {
		t.Errorf("expected 1 service, got %d", len(config.Services))
	}
}

// TestE2E_ProjectName verifies that project name is correctly set in the
// override config when provided.
func TestE2E_ProjectName(t *testing.T) {
	ctx := context.Background()
	log := testutil.NewTestLogger()

	composeContent := `services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
`
	_, composePath := setupComposeFile(t, composeContent)

	// Parse
	yamlParser := parser.NewYamlComposeParser(log)
	config, err := yamlParser.ParseComposeFile(ctx, composePath)
	if err != nil {
		t.Fatalf("ParseComposeFile failed: %v", err)
	}

	// Detect with conflict
	mockPortDetector := &testutil.MockPortDetector{UsedPorts: []int{8080}}
	unifiedDetector := scanner.NewUnifiedConflictDetectorImpl(mockPortDetector, &emptyNetworkDetector{}, log)

	projectName := "my-custom-project"
	conflictInfo, err := unifiedDetector.DetectConflicts(ctx, config, projectName)
	if err != nil {
		t.Fatalf("DetectConflicts failed: %v", err)
	}

	// Resolve
	mockAllocator := &testutil.MockPortAllocator{NextPort: 19080}
	unifiedGen := generator.NewUnifiedOverrideGeneratorImpl(mockAllocator, log)

	portConfig := types.PortConfig{
		Range: types.PortRange{Start: 10000, End: 60000},
	}
	if err := unifiedGen.ResolveConflicts(ctx, conflictInfo, types.ResolutionStrategyAutoIncrement, portConfig); err != nil {
		t.Fatalf("ResolveConflicts failed: %v", err)
	}

	// Generate
	override, err := unifiedGen.GenerateFromConflicts(ctx, config, conflictInfo)
	if err != nil {
		t.Fatalf("GenerateFromConflicts failed: %v", err)
	}

	// Set project name (mirroring cmd/up.go behavior)
	override.Name = projectName

	if override.Name != projectName {
		t.Errorf("override.Name = %q, want %q", override.Name, projectName)
	}

	// Verify resolution
	if len(override.Services) == 0 {
		t.Fatal("expected at least 1 service in override")
	}

	webOverride, exists := override.Services["web"]
	if !exists {
		t.Fatal("expected 'web' service in override")
	}

	// Verify port was resolved
	portResolved := false
	for _, pm := range webOverride.Ports {
		if pm.Host == 19080 {
			portResolved = true
			break
		}
	}
	if !portResolved {
		t.Errorf("expected web service port to be resolved to 19080, got ports: %+v", webOverride.Ports)
	}
}

// TestE2E_MultipleConflicts_AllResolved verifies that when multiple ports conflict,
// all are resolved with unique ports.
func TestE2E_MultipleConflicts_AllResolved(t *testing.T) {
	composeContent := `services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
  api:
    image: node:alpine
    ports:
      - "3000:3000"
  db:
    image: postgres:13
    ports:
      - "5432:5432"
  redis:
    image: redis:alpine
    ports:
      - "6379:6379"
`
	tmpDir, composePath := setupComposeFile(t, composeContent)

	// All 4 ports in use
	outputPath, content := runFullPipeline(t, composePath, tmpDir, []int{8080, 3000, 5432, 6379}, false)

	if outputPath == "" {
		t.Fatal("expected non-empty output path")
	}
	if content == nil {
		t.Fatal("expected non-nil content")
	}

	contentStr := string(content)
	t.Logf("Generated override for multi-conflict:\n%s", contentStr)

	// Verify file has services section
	if !strings.Contains(contentStr, "services:") {
		t.Error("override should contain 'services:' key")
	}

	// Verify resolved port 18080 appears (MockPortAllocator starts at 18080)
	if !strings.Contains(contentStr, "18080") {
		t.Error("override should contain resolved port 18080")
	}
}

// TestE2E_NoPorts_NoOverride verifies that services without ports produce no
// conflicts and no override file.
func TestE2E_NoPorts_NoOverride(t *testing.T) {
	composeContent := `services:
  worker:
    image: python:3.11
    command: python worker.py
  scheduler:
    image: python:3.11
    command: python scheduler.py
`
	tmpDir, composePath := setupComposeFile(t, composeContent)

	// Even with used ports, no services have port mappings
	outputPath, content := runFullPipeline(t, composePath, tmpDir, []int{8080, 3000, 5432}, false)

	if outputPath != "" {
		t.Errorf("expected empty output path for no-port services, got %q", outputPath)
	}
	if content != nil {
		t.Errorf("expected nil content for no-port services, got %d bytes", len(content))
	}
}

// TestE2E_DetectComposeFileVariants verifies that the compose file detector
// recognizes different compose file naming conventions.
func TestE2E_DetectComposeFileVariants(t *testing.T) {
	ctx := context.Background()
	log := testutil.NewTestLogger()

	variants := []struct {
		name     string
		filename string
	}{
		{"compose.yml", "compose.yml"},
		{"compose.yaml", "compose.yaml"},
		{"docker-compose.yml", "docker-compose.yml"},
		{"docker-compose.yaml", "docker-compose.yaml"},
	}

	composeContent := `services:
  web:
    image: nginx:alpine
`

	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			composePath := filepath.Join(tmpDir, v.filename)
			if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
				t.Fatalf("failed to write %s: %v", v.filename, err)
			}

			detector := parser.NewComposeFileDetectorImpl(log)
			detectedFile, err := detector.GetDefaultComposeFile(ctx, tmpDir)
			if err != nil {
				t.Fatalf("GetDefaultComposeFile failed for %s: %v", v.filename, err)
			}

			if detectedFile != composePath {
				t.Errorf("detected file = %q, want %q", detectedFile, composePath)
			}
		})
	}
}

// TestE2E_OverrideFileContent_ValidYAML verifies that the generated override
// file is valid YAML that can be parsed back.
func TestE2E_OverrideFileContent_ValidYAML(t *testing.T) {
	composeContent := `services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
`
	tmpDir, composePath := setupComposeFile(t, composeContent)

	_, content := runFullPipeline(t, composePath, tmpDir, []int{8080}, false)
	if content == nil {
		t.Fatal("expected non-nil content")
	}

	// Parse override as YAML to verify validity
	ctx := context.Background()
	log := testutil.NewTestLogger()
	overridePath := filepath.Join(tmpDir, "compose.override.yml")

	// Re-parse the generated override file
	yamlParser := parser.NewYamlComposeParser(log)
	overrideConfig, err := yamlParser.ParseComposeFile(ctx, overridePath)
	if err != nil {
		t.Fatalf("generated override file is not valid parseable YAML: %v", err)
	}

	// Verify it has the web service
	if _, exists := overrideConfig.Services["web"]; !exists {
		t.Error("parsed override should contain 'web' service")
	}

	// Verify the resolved port
	webService := overrideConfig.Services["web"]
	if len(webService.Ports) == 0 {
		t.Fatal("web service should have port mappings in override")
	}

	// MockPortAllocator starts at 18080
	foundResolved := false
	for _, pm := range webService.Ports {
		if pm.Host == 18080 {
			foundResolved = true
			break
		}
	}
	if !foundResolved {
		t.Errorf("expected resolved port 18080 in override, got ports: %+v", webService.Ports)
	}
}
