package config

import (
	"reflect"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// verify port config
	if config.Port.Range.Start != 8000 {
		t.Errorf("Port.Range.Start = %d, want 8000", config.Port.Range.Start)
	}
	if config.Port.Range.End != 9999 {
		t.Errorf("Port.Range.End = %d, want 9999", config.Port.Range.End)
	}
	if !config.Port.ExcludePrivileged {
		t.Error("Port.ExcludePrivileged = false, want true")
	}

	// verify file config
	if config.File.ComposeFile != "compose.yml" {
		t.Errorf("File.ComposeFile = %s, want compose.yml", config.File.ComposeFile)
	}
	if config.File.OverrideFile != "compose.override.yml" {
		t.Errorf("File.OverrideFile = %s, want compose.override.yml", config.File.OverrideFile)
	}

	// verify log config
	if config.Log.Level != "warn" {
		t.Errorf("Log.Level = %s, want warn", config.Log.Level)
	}
	if config.Log.Format != "text" {
		t.Errorf("Log.Format = %s, want text", config.Log.Format)
	}
}

func TestDefaultPortConfig(t *testing.T) {
	config := DefaultPortConfig()

	if config.Range.Start != 8000 {
		t.Errorf("Range.Start = %d, want 8000", config.Range.Start)
	}
	if config.Range.End != 9999 {
		t.Errorf("Range.End = %d, want 9999", config.Range.End)
	}

	expectedReserved := []int{8080, 8443, 9000, 9090}
	if !reflect.DeepEqual(config.Reserved, expectedReserved) {
		t.Errorf("Reserved = %v, want %v", config.Reserved, expectedReserved)
	}

	if !config.ExcludePrivileged {
		t.Error("ExcludePrivileged = false, want true")
	}
}

func TestDefaultFileConfig(t *testing.T) {
	config := DefaultFileConfig()

	if config.ComposeFile != "compose.yml" {
		t.Errorf("ComposeFile = %s, want compose.yml", config.ComposeFile)
	}
	if config.OverrideFile != "compose.override.yml" {
		t.Errorf("OverrideFile = %s, want compose.override.yml", config.OverrideFile)
	}
}

func TestDefaultLogConfig(t *testing.T) {
	config := DefaultLogConfig()

	if config.Level != "warn" {
		t.Errorf("Level = %s, want warn", config.Level)
	}
	if config.Format != "text" {
		t.Errorf("Format = %s, want text", config.Format)
	}
}

func TestDevelopmentConfig(t *testing.T) {
	config := DevelopmentConfig()

	if config.Log.Level != "debug" {
		t.Errorf("Log.Level = %s, want debug", config.Log.Level)
	}
	if config.Log.Format != "text" {
		t.Errorf("Log.Format = %s, want text", config.Log.Format)
	}

	// verify base config is correctly inherited
	if config.Port.Range.Start != 8000 {
		t.Errorf("Port.Range.Start = %d, want 8000", config.Port.Range.Start)
	}
}

func TestProductionConfig(t *testing.T) {
	config := ProductionConfig()

	if config.Log.Level != "warn" {
		t.Errorf("Log.Level = %s, want warn", config.Log.Level)
	}
	if config.Log.Format != "json" {
		t.Errorf("Log.Format = %s, want json", config.Log.Format)
	}
	if config.Log.File != "/var/log/gopose/gopose.log" {
		t.Errorf("Log.File = %s, want /var/log/gopose/gopose.log", config.Log.File)
	}

	// verify base config is correctly inherited
	if config.Port.Range.Start != 8000 {
		t.Errorf("Port.Range.Start = %d, want 8000", config.Port.Range.Start)
	}
}

func TestTestConfig(t *testing.T) {
	config := TestConfig()

	if config.Log.Level != "debug" {
		t.Errorf("Log.Level = %s, want debug", config.Log.Level)
	}
	if config.Log.Format != "text" {
		t.Errorf("Log.Format = %s, want text", config.Log.Format)
	}

	// verify base config is correctly inherited
	if config.Port.Range.Start != 8000 {
		t.Errorf("Port.Range.Start = %d, want 8000", config.Port.Range.Start)
	}
}

func TestConfigIsolation(t *testing.T) {
	// verify each config function returns an independent instance
	config1 := DefaultConfig()
	config2 := DefaultConfig()

	// verify pointers are different (== on *Config compares addresses, not values)
	if config1 == config2 {
		t.Error("DefaultConfig() returns the same pointer, expected different instances")
	}

	// verify modifying one does not affect the other
	config1.Port.Range.Start = 9999
	if config2.Port.Range.Start == 9999 {
		t.Error("Modifying config1 affected config2, expected independence")
	}
}
