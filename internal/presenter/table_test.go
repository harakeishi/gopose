package presenter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/harakeishi/gopose/pkg/types"
)

func TestTablePresenter_Progress(t *testing.T) {
	var buf bytes.Buffer
	p := NewTablePresenter(&buf)

	p.Progress("Scanning...")

	got := buf.String()
	if got != "Scanning...\n" {
		t.Errorf("Progress() = %q, want %q", got, "Scanning...\n")
	}
}

func TestTablePresenter_PortConflicts_WithConflicts(t *testing.T) {
	var buf bytes.Buffer
	p := NewTablePresenter(&buf)

	conflicts := []types.PortConflictInfo{
		{
			ServiceName: "web",
			Port:        3000,
			Resolution:  &types.PortResolutionInfo{ResolvedPort: 8001},
		},
		{
			ServiceName: "api",
			Port:        5432,
			Resolution:  &types.PortResolutionInfo{ResolvedPort: 5433},
		},
	}

	p.PortConflicts(conflicts)

	got := buf.String()
	if !strings.Contains(got, "Port Conflicts:") {
		t.Error("expected header 'Port Conflicts:'")
	}
	if !strings.Contains(got, "SERVICE") {
		t.Error("expected column header 'SERVICE'")
	}
	if !strings.Contains(got, "web") {
		t.Error("expected service name 'web'")
	}
	if !strings.Contains(got, "3000") {
		t.Error("expected port '3000'")
	}
	if !strings.Contains(got, "8001") {
		t.Error("expected resolved port '8001'")
	}
	if !strings.Contains(got, "api") {
		t.Error("expected service name 'api'")
	}
}

func TestTablePresenter_PortConflicts_Empty(t *testing.T) {
	var buf bytes.Buffer
	p := NewTablePresenter(&buf)

	p.PortConflicts(nil)

	got := buf.String()
	if !strings.Contains(got, "Port Conflicts:") {
		t.Error("expected header 'Port Conflicts:'")
	}
	if !strings.Contains(got, "(none)") {
		t.Error("expected '(none)' for empty conflicts")
	}
}

func TestTablePresenter_NetworkConflicts_WithConflicts(t *testing.T) {
	var buf bytes.Buffer
	p := NewTablePresenter(&buf)

	conflicts := []types.NetworkConflictInfo{
		{
			NetworkName:    "default",
			OriginalSubnet: "172.20.0.0/24",
			Resolution:     &types.NetworkResolutionInfo{ResolvedSubnet: "10.20.0.0/24"},
		},
	}

	p.NetworkConflicts(conflicts)

	got := buf.String()
	if !strings.Contains(got, "Network Conflicts:") {
		t.Error("expected header 'Network Conflicts:'")
	}
	if !strings.Contains(got, "default") {
		t.Error("expected network name 'default'")
	}
	if !strings.Contains(got, "172.20.0.0/24") {
		t.Error("expected original subnet")
	}
	if !strings.Contains(got, "10.20.0.0/24") {
		t.Error("expected resolved subnet")
	}
}

func TestTablePresenter_NetworkConflicts_Empty(t *testing.T) {
	var buf bytes.Buffer
	p := NewTablePresenter(&buf)

	p.NetworkConflicts(nil)

	got := buf.String()
	if !strings.Contains(got, "Network Conflicts:") {
		t.Error("expected header 'Network Conflicts:'")
	}
	if !strings.Contains(got, "(none)") {
		t.Error("expected '(none)' for empty conflicts")
	}
}

func TestTablePresenter_Result(t *testing.T) {
	var buf bytes.Buffer
	p := NewTablePresenter(&buf)

	p.Result("Generated: compose.override.yml")

	got := buf.String()
	if got != "\nGenerated: compose.override.yml\n" {
		t.Errorf("Result() = %q, want %q", got, "\nGenerated: compose.override.yml\n")
	}
}

func TestTablePresenter_PortConflicts_NoResolution(t *testing.T) {
	var buf bytes.Buffer
	p := NewTablePresenter(&buf)

	conflicts := []types.PortConflictInfo{
		{
			ServiceName: "web",
			Port:        3000,
			Resolution:  nil,
		},
	}

	p.PortConflicts(conflicts)

	got := buf.String()
	// Conflicts without resolution should be skipped, showing (none)
	if !strings.Contains(got, "(none)") {
		t.Error("conflicts without resolution should show (none)")
	}
}

func TestTablePresenter_FullWorkflow(t *testing.T) {
	var buf bytes.Buffer
	p := NewTablePresenter(&buf)

	p.Progress("Scanning...")
	p.Progress("Resolving...")
	p.PortConflicts([]types.PortConflictInfo{
		{
			ServiceName: "web",
			Port:        3000,
			Resolution:  &types.PortResolutionInfo{ResolvedPort: 8001},
		},
	})
	p.NetworkConflicts(nil)
	p.Result("Generated: compose.override.yml")

	got := buf.String()
	scanIdx := strings.Index(got, "Scanning...")
	resolveIdx := strings.Index(got, "Resolving...")
	portIdx := strings.Index(got, "Port Conflicts:")
	netIdx := strings.Index(got, "Network Conflicts:")
	genIdx := strings.Index(got, "Generated:")

	if scanIdx >= resolveIdx {
		t.Error("Scanning should come before Resolving")
	}
	if resolveIdx >= portIdx {
		t.Error("Resolving should come before Port Conflicts")
	}
	if portIdx >= netIdx {
		t.Error("Port Conflicts should come before Network Conflicts")
	}
	if netIdx >= genIdx {
		t.Error("Network Conflicts should come before Generated")
	}
}
