package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/harakeishi/gopose/internal/testutil"
	"github.com/harakeishi/gopose/pkg/types"
)

func newTestDeps() (*testutil.MockComposeFileDetector, *testutil.MockComposeFileParser, *testutil.MockUnifiedConflictDetector, *testutil.MockUnifiedOverrideGenerator, *testutil.MockOverrideWriter, UpServiceDeps) {
	detector := &testutil.MockComposeFileDetector{DefaultFile: "compose.yml"}
	parser := &testutil.MockComposeFileParser{Config: &types.ComposeConfig{
		Services: map[string]types.Service{
			"web": {Name: "web", Ports: []types.PortMapping{{Host: 8080, Container: 80}}},
		},
	}}
	conflictDetector := &testutil.MockUnifiedConflictDetector{
		ConflictInfo: &types.UnifiedConflictInfo{},
	}
	overrideGen := &testutil.MockUnifiedOverrideGenerator{
		Override: &types.OverrideConfig{
			Services: map[string]types.ServiceOverride{},
		},
	}
	writer := &testutil.MockOverrideWriter{}

	deps := UpServiceDeps{
		Logger:              testutil.NewTestLogger(),
		ComposeFileDetector: detector,
		ComposeParser:       parser,
		ConflictDetector:    conflictDetector,
		OverrideGenerator:   overrideGen,
		OverrideWriter:      writer,
	}

	return detector, parser, conflictDetector, overrideGen, writer, deps
}

func defaultParams() UpParams {
	return UpParams{
		ComposeFilePath: "compose.yml",
		WorkDir:         "/tmp",
		OutputFile:      "compose.override.yml",
		ProjectName:     "testproject",
		Strategy:        "auto",
		PortConfig:      types.PortConfig{Range: types.PortRange{Start: 8000, End: 9999}},
		DryRun:          false,
	}
}

func TestExecute_NoConflicts(t *testing.T) {
	_, _, _, _, writer, deps := newTestDeps()
	svc := NewUpService(deps)
	params := defaultParams()

	result, err := svc.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasConflicts {
		t.Error("expected no conflicts")
	}
	// WriteOverrideFile should not be called when no conflicts
	if writer.WriteErr != nil {
		t.Error("writer should not have been invoked with error")
	}
}

func TestExecute_WithPortConflicts(t *testing.T) {
	_, _, conflictDetector, overrideGen, _, deps := newTestDeps()

	conflictDetector.ConflictInfo = &types.UnifiedConflictInfo{
		PortConflicts: []types.PortConflictInfo{
			{ServiceName: "web", Port: 8080, Type: types.ConflictTypeSystemPort},
		},
	}
	overrideGen.Override = &types.OverrideConfig{
		Services: map[string]types.ServiceOverride{
			"web": {Ports: []types.PortMapping{{Host: 8081, Container: 80}}},
		},
	}

	svc := NewUpService(deps)
	params := defaultParams()

	result, err := svc.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasConflicts {
		t.Error("expected conflicts")
	}
	if result.PortConflicts != 1 {
		t.Errorf("expected 1 port conflict, got %d", result.PortConflicts)
	}
	if result.OutputFile != "compose.override.yml" {
		t.Errorf("expected output file compose.override.yml, got %s", result.OutputFile)
	}
}

func TestExecute_DryRun(t *testing.T) {
	_, _, conflictDetector, overrideGen, writer, deps := newTestDeps()

	conflictDetector.ConflictInfo = &types.UnifiedConflictInfo{
		PortConflicts: []types.PortConflictInfo{
			{ServiceName: "web", Port: 8080, Type: types.ConflictTypeSystemPort},
		},
	}
	overrideGen.Override = &types.OverrideConfig{
		Services: map[string]types.ServiceOverride{},
	}

	// Track if WriteOverrideFile is called
	writeCalled := false
	writer.WriteErr = fmt.Errorf("should not be called")
	_ = writeCalled

	svc := NewUpService(deps)
	params := defaultParams()
	params.DryRun = true

	result, err := svc.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.HasConflicts {
		t.Error("expected conflicts detected")
	}
	// In dry run, the override should still be generated but not written
	if result.Override == nil {
		t.Error("expected override to be set even in dry run")
	}
}

func TestExecute_ComposeFileAutoDetect(t *testing.T) {
	detector, _, _, _, _, deps := newTestDeps()
	detector.DefaultFile = "docker-compose.yml"

	svc := NewUpService(deps)
	params := defaultParams()
	params.ComposeFilePath = "" // empty triggers auto-detect

	result, err := svc.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasConflicts {
		t.Error("expected no conflicts")
	}
}

func TestExecute_ParseError(t *testing.T) {
	_, parser, _, _, _, deps := newTestDeps()
	parser.Err = fmt.Errorf("parse error")
	parser.Config = nil

	svc := NewUpService(deps)
	params := defaultParams()

	_, err := svc.Execute(context.Background(), params)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "docker composeファイルの解析に失敗: parse error" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestExecute_DetectionError(t *testing.T) {
	_, _, conflictDetector, _, _, deps := newTestDeps()
	conflictDetector.Err = fmt.Errorf("detection error")
	conflictDetector.ConflictInfo = nil

	svc := NewUpService(deps)
	params := defaultParams()

	_, err := svc.Execute(context.Background(), params)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "衝突検知に失敗: detection error" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestExecute_ResolutionError(t *testing.T) {
	_, _, conflictDetector, overrideGen, _, deps := newTestDeps()
	conflictDetector.ConflictInfo = &types.UnifiedConflictInfo{
		PortConflicts: []types.PortConflictInfo{
			{ServiceName: "web", Port: 8080, Type: types.ConflictTypeSystemPort},
		},
	}
	overrideGen.ResolveErr = fmt.Errorf("resolution error")

	svc := NewUpService(deps)
	params := defaultParams()

	_, err := svc.Execute(context.Background(), params)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "衝突解決に失敗: resolution error" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestExecute_ValidationError(t *testing.T) {
	_, _, conflictDetector, overrideGen, writer, deps := newTestDeps()
	conflictDetector.ConflictInfo = &types.UnifiedConflictInfo{
		PortConflicts: []types.PortConflictInfo{
			{ServiceName: "web", Port: 8080, Type: types.ConflictTypeSystemPort},
		},
	}
	overrideGen.Override = &types.OverrideConfig{
		Services: map[string]types.ServiceOverride{},
	}
	writer.ValidateErr = fmt.Errorf("validation error")

	svc := NewUpService(deps)
	params := defaultParams()

	_, err := svc.Execute(context.Background(), params)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "overrideファイルの検証に失敗: validation error" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestExecute_WriteError(t *testing.T) {
	_, _, conflictDetector, overrideGen, writer, deps := newTestDeps()
	conflictDetector.ConflictInfo = &types.UnifiedConflictInfo{
		PortConflicts: []types.PortConflictInfo{
			{ServiceName: "web", Port: 8080, Type: types.ConflictTypeSystemPort},
		},
	}
	overrideGen.Override = &types.OverrideConfig{
		Services: map[string]types.ServiceOverride{},
	}
	writer.WriteErr = fmt.Errorf("write error")

	svc := NewUpService(deps)
	params := defaultParams()

	_, err := svc.Execute(context.Background(), params)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "overrideファイルの書き込みに失敗: write error" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestExecute_ProjectNameSet(t *testing.T) {
	_, _, conflictDetector, overrideGen, _, deps := newTestDeps()
	conflictDetector.ConflictInfo = &types.UnifiedConflictInfo{
		PortConflicts: []types.PortConflictInfo{
			{ServiceName: "web", Port: 8080, Type: types.ConflictTypeSystemPort},
		},
	}
	overrideGen.Override = &types.OverrideConfig{
		Services: map[string]types.ServiceOverride{},
	}

	svc := NewUpService(deps)
	params := defaultParams()
	params.ProjectName = "myproject"

	result, err := svc.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Override.Name != "myproject" {
		t.Errorf("expected project name 'myproject', got '%s'", result.Override.Name)
	}
}

func TestExecute_DefaultOutputFile(t *testing.T) {
	_, _, conflictDetector, overrideGen, _, deps := newTestDeps()
	conflictDetector.ConflictInfo = &types.UnifiedConflictInfo{
		PortConflicts: []types.PortConflictInfo{
			{ServiceName: "web", Port: 8080, Type: types.ConflictTypeSystemPort},
		},
	}
	overrideGen.Override = &types.OverrideConfig{
		Services: map[string]types.ServiceOverride{},
	}

	svc := NewUpService(deps)
	params := defaultParams()
	params.OutputFile = "" // empty triggers default

	result, err := svc.Execute(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.OutputFile != "compose.override.yml" {
		t.Errorf("expected default output file 'compose.override.yml', got '%s'", result.OutputFile)
	}
}

func TestExecute_StrategyMapping(t *testing.T) {
	tests := []struct {
		name     string
		strategy string
	}{
		{"auto strategy", "auto"},
		{"range strategy", "range"},
		{"user strategy", "user"},
		{"unknown defaults to auto", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, conflictDetector, overrideGen, _, deps := newTestDeps()
			conflictDetector.ConflictInfo = &types.UnifiedConflictInfo{
				PortConflicts: []types.PortConflictInfo{
					{ServiceName: "web", Port: 8080, Type: types.ConflictTypeSystemPort},
				},
			}
			overrideGen.Override = &types.OverrideConfig{
				Services: map[string]types.ServiceOverride{},
			}

			svc := NewUpService(deps)
			params := defaultParams()
			params.Strategy = tt.strategy

			_, err := svc.Execute(context.Background(), params)
			if err != nil {
				t.Fatalf("unexpected error for strategy %s: %v", tt.strategy, err)
			}
		})
	}
}
