package scanner

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/harakeishi/gopose/internal/logger"
)

// NetworkInfo holds basic information about an existing Docker network.
type NetworkInfo struct {
	Name    string   `json:"Name"`
	Subnet  string   `json:"Subnet"`
	Subnets []string `json:"Subnets"`
}

// NetworkDetector はネットワーク検出のインターフェースです。
type NetworkDetector interface {
	DetectNetworks(ctx context.Context) ([]NetworkInfo, error)
}

// DockerNetworkDetector detects existing Docker networks and their subnets.
type DockerNetworkDetector struct {
	logger logger.Logger
}

// NewDockerNetworkDetector creates a new detector.
func NewDockerNetworkDetector(l logger.Logger) *DockerNetworkDetector {
	return &DockerNetworkDetector{logger: l}
}

// DetectNetworks returns current Docker networks and their subnets.
func (d *DockerNetworkDetector) DetectNetworks(ctx context.Context) ([]NetworkInfo, error) {
	// List network IDs
	out, err := exec.CommandContext(ctx, "docker", "network", "ls", "-q").Output()
	if err != nil {
		return nil, err
	}
	ids := strings.Fields(strings.TrimSpace(string(out)))
	var networks []NetworkInfo
	for _, id := range ids {
		inspectOut, err := exec.CommandContext(ctx, "docker", "network", "inspect", id, "--format", "{{json .}}").Output()
		if err != nil {
			continue // skip network on error
		}
		var raw struct {
			Name string `json:"Name"`
			IPAM struct {
				Config []struct {
					Subnet string `json:"Subnet"`
				} `json:"Config"`
			} `json:"IPAM"`
		}
		if err := json.Unmarshal(inspectOut, &raw); err != nil {
			continue
		}
		var subs []string
		for _, cfg := range raw.IPAM.Config {
			if cfg.Subnet != "" {
				subs = append(subs, cfg.Subnet)
			}
		}
		var primarySubnet string
		if len(subs) > 0 {
			primarySubnet = subs[0]
		}
		networks = append(networks, NetworkInfo{Name: raw.Name, Subnet: primarySubnet, Subnets: subs})
	}
	return networks, nil
}
