package generator

import (
    "context"
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/harakeishi/gopose/internal/errors"
    "github.com/harakeishi/gopose/internal/logger"
    "github.com/harakeishi/gopose/pkg/types"
)

var nopLogger = &logger.NopLogger{}

func TestGenerateOverrideBuildsServices(t *testing.T) {
    generator := NewOverrideGeneratorImpl(nopLogger)
    ctx := context.Background()

    config := &types.ComposeConfig{
        Version: "3",
        Services: map[string]types.Service{
            "web": {
                Ports: []types.PortMapping{{Host: 8080, Container: 80}},
            },
        },
    }
    resolutions := []types.ConflictResolution{{ServiceName: "web", ConflictPort: 8080, ResolvedPort: 9000}}

    override, err := generator.GenerateOverride(ctx, config, resolutions)
    if err != nil {
        t.Fatalf("GenerateOverride returned error: %v", err)
    }
    svc, ok := override.Services["web"]
    if !ok {
        t.Fatalf("expected override for web service")
    }
    if svc.Ports[0].Host != 9000 {
        t.Fatalf("expected host port to be updated, got %d", svc.Ports[0].Host)
    }
    if override.Metadata.GeneratedAt.IsZero() {
        t.Fatalf("expected metadata timestamp to be set")
    }
}

func TestGenerateServiceOverrideErrors(t *testing.T) {
    generator := NewOverrideGeneratorImpl(nopLogger)
    ctx := context.Background()
    config := &types.ComposeConfig{Services: map[string]types.Service{}}

    _, err := generator.generateServiceOverride(ctx, "missing", nil, config)
    if err == nil {
        t.Fatalf("expected error for missing service")
    }
    if appErr, ok := err.(*errors.AppError); !ok || appErr.Code != errors.ErrValidationFailed {
        t.Fatalf("expected validation error, got %#v", err)
    }
}

func TestGenerateServiceOverrideResolvesPorts(t *testing.T) {
    generator := NewOverrideGeneratorImpl(nopLogger)
    ctx := context.Background()
    config := &types.ComposeConfig{Services: map[string]types.Service{
        "api": {Ports: []types.PortMapping{{Host: 8080, Container: 8080}}},
    }}
    resolutions := []types.ConflictResolution{{ServiceName: "api", ConflictPort: 8080, ResolvedPort: 9001}}

    override, err := generator.generateServiceOverride(ctx, "api", resolutions, config)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if override.Ports[0].Host != 9001 {
        t.Fatalf("expected host to be replaced, got %d", override.Ports[0].Host)
    }
}

func TestValidateServiceOverride(t *testing.T) {
    generator := NewOverrideGeneratorImpl(nopLogger)
    ctx := context.Background()

    override := types.ServiceOverride{
        Ports: []types.PortMapping{
            {Host: 8080, Container: 80},
            {Host: 8080, Container: 8080},
        },
    }
    if err := generator.validateServiceOverride(ctx, "web", override); err == nil {
        t.Fatalf("expected duplicate host port error")
    }

    invalid := types.ServiceOverride{
        Ports: []types.PortMapping{{Host: -1, Container: 80}},
    }
    if err := generator.validateServiceOverride(ctx, "web", invalid); err == nil {
        t.Fatalf("expected invalid host port error")
    }
}

func TestValidateResolutionUniqueness(t *testing.T) {
    generator := NewOverrideGeneratorImpl(nopLogger)
    ctx := context.Background()

    resolutions := []types.ConflictResolution{
        {ServiceName: "api", ResolvedPort: 9000},
        {ServiceName: "web", ResolvedPort: 9000},
    }
    if err := generator.validateResolutionUniqueness(ctx, resolutions); err == nil {
        t.Fatalf("expected duplication error")
    }

    if err := generator.validateResolutionUniqueness(ctx, []types.ConflictResolution{}); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

func TestGenerateOverrideYAML(t *testing.T) {
    generator := NewOverrideGeneratorImpl(nopLogger)
    override := &types.OverrideConfig{
        Name: "example",
        Services: map[string]types.ServiceOverride{
            "web": {Ports: []types.PortMapping{{Host: 9000, Container: 80}}},
        },
    }

    yaml := generator.generateOverrideYAML(override)
    if !strings.Contains(yaml, "ports: !override") {
        t.Fatalf("expected !override tag in yaml: %s", yaml)
    }
    if !strings.Contains(yaml, "name: example") {
        t.Fatalf("expected project name in yaml")
    }
}

func TestGenerateFileHeader(t *testing.T) {
    generator := NewOverrideGeneratorImpl(nopLogger)
    header := generator.generateFileHeader()
    if !strings.Contains(header, "Docker Compose Override File") {
        t.Fatalf("expected header to mention override file")
    }
    if !strings.Contains(header, "WARNING") {
        t.Fatalf("expected warning in header")
    }
}

func TestWriteOverrideFile(t *testing.T) {
    generator := NewOverrideGeneratorImpl(nopLogger)
    ctx := context.Background()
    override := &types.OverrideConfig{Services: map[string]types.ServiceOverride{}}

    dir := t.TempDir()
    path := filepath.Join(dir, "override.yml")
    if err := generator.WriteOverrideFile(ctx, override, path); err != nil {
        t.Fatalf("unexpected error writing override file: %v", err)
    }

    data, err := os.ReadFile(path)
    if err != nil {
        t.Fatalf("failed to read generated file: %v", err)
    }
    if !strings.Contains(string(data), "Docker Compose Override File") {
        t.Fatalf("generated file missing header")
    }
}

func TestGenerateFromTemplate(t *testing.T) {
    tmpl := "services:\n  web:\n    ports:\n      - host: 8080\n        container: 80\n"
    dir := t.TempDir()
    path := filepath.Join(dir, "template.yaml")
    if err := os.WriteFile(path, []byte(tmpl), 0644); err != nil {
        t.Fatalf("write template: %v", err)
    }

    generator := NewOverrideTemplateGeneratorImpl(nopLogger)
    cfg, err := generator.GenerateFromTemplate(context.Background(), path, nil)
    if err != nil {
        t.Fatalf("GenerateFromTemplate: %v", err)
    }
    if _, ok := cfg.Services["web"]; !ok {
        t.Fatalf("expected web service in override")
    }
}

func TestValidateTemplate(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "template.yml")
    if err := os.WriteFile(path, []byte("services: {}"), 0644); err != nil {
        t.Fatalf("write template: %v", err)
    }

    generator := NewOverrideTemplateGeneratorImpl(nopLogger)
    if err := generator.ValidateTemplate(context.Background(), path); err != nil {
        t.Fatalf("ValidateTemplate returned error: %v", err)
    }
}

func TestValidateOverrideWithNoServices(t *testing.T) {
    generator := NewOverrideGeneratorImpl(nopLogger)
    ctx := context.Background()
    override := &types.OverrideConfig{Services: map[string]types.ServiceOverride{}}
    if err := generator.ValidateOverride(ctx, override); err != nil {
        t.Fatalf("expected no error for empty services: %v", err)
    }
}

func TestValidateServiceOverrideContainerRange(t *testing.T) {
    generator := NewOverrideGeneratorImpl(nopLogger)
    ctx := context.Background()
    override := types.ServiceOverride{Ports: []types.PortMapping{{Host: 9000, Container: 70000}}}
    if err := generator.validateServiceOverride(ctx, "web", override); err == nil {
        t.Fatalf("expected container port validation error")
    }
}
