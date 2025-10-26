package scanner

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/harakeishi/gopose/internal/logger"
)

func TestDockerNetworkDetector(t *testing.T) {
    dir := t.TempDir()
    script := "#!/bin/sh\nif [ \"$1\" = network ] && [ \"$2\" = ls ]; then\n  echo \"net1\nnet2\"\nelif [ \"$1\" = network ] && [ \"$2\" = inspect ]; then\n  id=$3\n  cat <<JSON\n{\"Name\":\"$id\",\"IPAM\":{\"Config\":[{\"Subnet\":\"10.0.0.0/24\"}]}}\nJSON\nelse\n  exit 1\nfi\n"
    path := filepath.Join(dir, "docker")
    if err := os.WriteFile(path, []byte(script), 0755); err != nil {
        t.Fatalf("write script: %v", err)
    }
    t.Setenv("PATH", dir+":"+os.Getenv("PATH"))

    detector := NewDockerNetworkDetector(&logger.NopLogger{})
    networks, err := detector.DetectNetworks(context.Background())
    if err != nil {
        t.Fatalf("DetectNetworks error: %v", err)
    }
    if len(networks) != 2 {
        t.Fatalf("expected two networks, got %d", len(networks))
    }
    if networks[0].Name == "" || len(networks[0].Subnets) == 0 {
        t.Fatalf("expected network details: %#v", networks[0])
    }
}
