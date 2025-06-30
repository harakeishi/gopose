package types

import "time"

// Service はDocker Composeサービスを表します。
type Service struct {
	Name        string                     `yaml:"name" json:"name"`
	Image       string                     `yaml:"image" json:"image"`
	Ports       []PortMapping              `yaml:"ports" json:"ports"`
	DependsOn   []string                   `yaml:"depends_on" json:"depends_on"`
	Environment map[string]string          `yaml:"environment" json:"environment"`
	Networks    map[string]ServiceNetwork  `yaml:"networks" json:"networks"`
}

// ComposeConfig はDocker Composeファイルの設定を表します。
type ComposeConfig struct {
	Version  string             `yaml:"version" json:"version"`
	Services map[string]Service `yaml:"services" json:"services"`
	Networks map[string]Network `yaml:"networks" json:"networks"`
	Volumes  map[string]Volume  `yaml:"volumes" json:"volumes"`
	FilePath string             `yaml:"-" json:"file_path"`
}

// Network はDocker Composeネットワーク設定を表します。
type Network struct {
	Driver string            `yaml:"driver" json:"driver"`
	IPAM   IPAM              `yaml:"ipam" json:"ipam"`
	Labels map[string]string `yaml:"labels" json:"labels"`
}

// IPAM はIPアドレス管理設定を表します。
type IPAM struct {
	Driver string       `yaml:"driver" json:"driver"`
	Config []IPAMConfig `yaml:"config" json:"config"`
}

// IPAMConfig はIPAM設定の詳細を表します。
type IPAMConfig struct {
	Subnet  string `yaml:"subnet" json:"subnet"`
	Gateway string `yaml:"gateway" json:"gateway"`
}

// Volume はDocker Composeボリューム設定を表します。
type Volume struct {
	Driver     string            `yaml:"driver" json:"driver"`
	DriverOpts map[string]string `yaml:"driver_opts" json:"driver_opts"`
	Labels     map[string]string `yaml:"labels" json:"labels"`
}

// OverrideConfig はoverride.ymlファイルの構造を表します。
type OverrideConfig struct {
	Version  string                     `yaml:"version" json:"version"`
	Services map[string]ServiceOverride `yaml:"services" json:"services"`
	Networks map[string]NetworkOverride `yaml:"networks,omitempty" json:"networks,omitempty"`
	Metadata OverrideMetadata           `yaml:"x-gopose-metadata" json:"metadata"`
}

// ServiceOverride はサービスのオーバーライド設定を表します。
type ServiceOverride struct {
	Ports    []PortMapping              `yaml:"ports" json:"ports"`
	Networks map[string]ServiceNetwork  `yaml:"networks" json:"networks"`
}

// ServiceNetwork はサービスのネットワーク設定を表します。
type ServiceNetwork struct {
	IPv4Address string `yaml:"ipv4_address,omitempty" json:"ipv4_address,omitempty"`
}

// OverrideMetadata は生成情報とメタデータを表します。
type OverrideMetadata struct {
	GeneratedAt time.Time            `yaml:"generated_at" json:"generated_at"`
	Version     string               `yaml:"version" json:"version"`
	Resolutions []ConflictResolution `yaml:"resolutions" json:"resolutions"`
}

// NetworkOverride はネットワーク設定のオーバーライドを表します。
// 現状は subnet だけを上書き対象とする。
type NetworkOverride struct {
	Driver string            `yaml:"driver,omitempty" json:"driver,omitempty"`
	IPAM   IPAM              `yaml:"ipam,omitempty" json:"ipam,omitempty"`
	Labels map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}
