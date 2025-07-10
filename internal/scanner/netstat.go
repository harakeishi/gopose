package scanner

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/harakeishi/gopose/internal/errors"
	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/pkg/types"
)

// NetstatPortDetector はnetstatコマンドを使用したポート検出実装です。
type NetstatPortDetector struct {
	logger logger.Logger
}

// NewNetstatPortDetector は新しいNetstatPortDetectorを作成します。
func NewNetstatPortDetector(logger logger.Logger) *NetstatPortDetector {
	return &NetstatPortDetector{
		logger: logger,
	}
}

// DetectUsedPorts はシステムで使用中のポートを検出します。
func (n *NetstatPortDetector) DetectUsedPorts(ctx context.Context) ([]int, error) {
	n.logger.Debug(ctx, "netstatを使用してポートスキャンを開始")

	// netstatコマンドを実行（macOS対応）
	cmd := exec.CommandContext(ctx, "netstat", "-an")
	output, err := cmd.Output()
	if err != nil {
		return nil, &errors.AppError{
			Code:    errors.ErrPortScanFailed,
			Message: "netstatコマンドの実行に失敗しました",
			Cause:   err,
		}
	}

	ports, err := n.parseNetstatOutput(string(output))
	if err != nil {
		return nil, err
	}

	sort.Ints(ports)
	n.logger.Info(ctx, "ポートスキャン完了",
		types.Field{Key: "found_ports_count", Value: len(ports)})

	return ports, nil
}

// DetectUsedPortsInRange は指定された範囲内の使用中ポートを検出します。
func (n *NetstatPortDetector) DetectUsedPortsInRange(ctx context.Context, portRange types.PortRange) ([]int, error) {
	allPorts, err := n.DetectUsedPorts(ctx)
	if err != nil {
		return nil, err
	}

	var portsInRange []int
	for _, port := range allPorts {
		if port >= portRange.Start && port <= portRange.End {
			portsInRange = append(portsInRange, port)
		}
	}

	n.logger.Debug(ctx, "範囲内ポートフィルタリング完了",
		types.Field{Key: "range_start", Value: portRange.Start},
		types.Field{Key: "range_end", Value: portRange.End},
		types.Field{Key: "filtered_count", Value: len(portsInRange)})

	return portsInRange, nil
}

// IsPortInUse は指定されたポートが使用中かどうかを確認します。
// 注意: このメソッドは個別ポートチェック用で、大量のポート確認には時間がかかります。
// 通常はDetectUsedPorts/DetectUsedPortsInRangeを使用することを推奨します。
func (n *NetstatPortDetector) IsPortInUse(ctx context.Context, port int) (bool, error) {
	// より効率的な方法として、直接そのポートに接続を試みる
	timeout := 100 * time.Millisecond

	// TCPポートをチェック
	tcpAddr := fmt.Sprintf("localhost:%d", port)
	tcpConn, err := net.DialTimeout("tcp", tcpAddr, timeout)
	if err == nil {
		tcpConn.Close()
		return true, nil
	}

	// UDPポートをチェック
	udpAddr := fmt.Sprintf("localhost:%d", port)
	udpConn, err := net.DialTimeout("udp", udpAddr, timeout)
	if err == nil {
		udpConn.Close()
		return true, nil
	}

	return false, nil
}

// parseNetstatOutput はnetstatの出力を解析してポート番号を抽出します。
func (n *NetstatPortDetector) parseNetstatOutput(output string) ([]int, error) {
	lines := strings.Split(output, "\n")
	ports := make(map[int]bool) // 重複を避けるためにmapを使用

	// macOS/BSD系のnetstat出力形式に対応する正規表現
	// 例1: tcp46      0      0  *.8080                 *.*                    LISTEN
	// 例2: tcp4       0      0  127.0.0.1.3333         *.*                    LISTEN
	re := regexp.MustCompile(`(?:tcp|udp)\S*\s+\d+\s+\d+\s+(?:\*|\d+\.\d+\.\d+\.\d+)\.(\d+)\s+.*LISTEN`)

	for _, line := range lines {
		// LISTENステートのみを対象とする
		if !strings.Contains(line, "LISTEN") {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) >= 2 {
			portStr := matches[1]
			port, err := strconv.Atoi(portStr)
			if err != nil {
				// ポート番号の変換に失敗した場合はスキップ
				continue
			}
			ports[port] = true
		}
	}

	// mapからスライスに変換
	result := make([]int, 0, len(ports))
	for port := range ports {
		result = append(result, port)
	}

	return result, nil
}

// PortAllocatorImpl はポート割り当ての実装です。
type PortAllocatorImpl struct {
	detector PortDetector
	logger   logger.Logger
}

// NewPortAllocatorImpl は新しいPortAllocatorImplを作成します。
func NewPortAllocatorImpl(detector PortDetector, logger logger.Logger) *PortAllocatorImpl {
	return &PortAllocatorImpl{
		detector: detector,
		logger:   logger,
	}
}

// AllocatePort は利用可能なポートを1つ割り当てます。
func (p *PortAllocatorImpl) AllocatePort(ctx context.Context, config types.PortConfig) (int, error) {
	usedPorts, err := p.detector.DetectUsedPortsInRange(ctx, config.Range)
	if err != nil {
		return 0, err
	}

	// 使用中ポートと予約済みポートを合わせた除外リスト
	excludePorts := make(map[int]bool)
	for _, port := range usedPorts {
		excludePorts[port] = true
	}
	for _, port := range config.Reserved {
		excludePorts[port] = true
	}

	// 特権ポートを除外
	if config.ExcludePrivileged {
		for i := 1; i <= 1023; i++ {
			excludePorts[i] = true
		}
	}

	// 利用可能なポートを順次検索（IsPortInUseでの個別チェックは削除）
	for port := config.Range.Start; port <= config.Range.End; port++ {
		if !excludePorts[port] {
			p.logger.Debug(ctx, "ポート割り当て成功",
				types.Field{Key: "allocated_port", Value: port})
			return port, nil
		}
	}

	return 0, &errors.AppError{
		Code:    errors.ErrPortUnavailable,
		Message: "指定された範囲に利用可能なポートがありません",
		Fields: map[string]interface{}{
			"range_start": config.Range.Start,
			"range_end":   config.Range.End,
		},
	}
}

// AllocatePorts は指定された数のポートを割り当てます。
func (p *PortAllocatorImpl) AllocatePorts(ctx context.Context, count int, config types.PortConfig) ([]int, error) {
	if count <= 0 {
		return []int{}, nil
	}

	// 一度だけ使用中ポートを取得
	usedPorts, err := p.detector.DetectUsedPortsInRange(ctx, config.Range)
	if err != nil {
		return nil, err
	}

	// 使用中ポートと予約済みポートを合わせた除外リスト
	excludePorts := make(map[int]bool)
	for _, port := range usedPorts {
		excludePorts[port] = true
	}
	for _, port := range config.Reserved {
		excludePorts[port] = true
	}

	// 特権ポートを除外
	if config.ExcludePrivileged {
		for i := 1; i <= 1023; i++ {
			excludePorts[i] = true
		}
	}

	allocatedPorts := make([]int, 0, count)

	// 利用可能なポートを順次検索
	for port := config.Range.Start; port <= config.Range.End && len(allocatedPorts) < count; port++ {
		if !excludePorts[port] {
			allocatedPorts = append(allocatedPorts, port)
			excludePorts[port] = true // 次の割り当てで除外
		}
	}

	if len(allocatedPorts) < count {
		return allocatedPorts, &errors.AppError{
			Code:    errors.ErrPortUnavailable,
			Message: fmt.Sprintf("要求された数のポートを割り当てできません。要求: %d, 割り当て可能: %d", count, len(allocatedPorts)),
			Fields: map[string]interface{}{
				"requested_count": count,
				"allocated_count": len(allocatedPorts),
				"range_start":     config.Range.Start,
				"range_end":       config.Range.End,
			},
		}
	}

	p.logger.Info(ctx, "複数ポート割り当て完了",
		types.Field{Key: "allocated_count", Value: len(allocatedPorts)},
		types.Field{Key: "ports", Value: allocatedPorts})

	return allocatedPorts, nil
}

// AllocatePortsForServices はサービス別にポートを割り当てます。
func (p *PortAllocatorImpl) AllocatePortsForServices(ctx context.Context, services []types.Service, config types.PortConfig) (map[string]int, error) {
	// ポートが必要なサービス数を計算
	servicesNeedingPorts := 0
	for _, service := range services {
		if len(service.Ports) > 0 {
			servicesNeedingPorts++
		}
	}

	if servicesNeedingPorts == 0 {
		return make(map[string]int), nil
	}

	// 一度だけ使用中ポートを取得
	usedPorts, err := p.detector.DetectUsedPortsInRange(ctx, config.Range)
	if err != nil {
		return nil, err
	}

	// 使用中ポートと予約済みポートを合わせた除外リスト
	excludePorts := make(map[int]bool)
	for _, port := range usedPorts {
		excludePorts[port] = true
	}
	for _, port := range config.Reserved {
		excludePorts[port] = true
	}

	// 特権ポートを除外
	if config.ExcludePrivileged {
		for i := 1; i <= 1023; i++ {
			excludePorts[i] = true
		}
	}

	result := make(map[string]int)
	currentPort := config.Range.Start

	for _, service := range services {
		if len(service.Ports) == 0 {
			continue // ポートマッピングがないサービスはスキップ
		}

		// 利用可能なポートを検索
		for currentPort <= config.Range.End {
			if !excludePorts[currentPort] {
				result[service.Name] = currentPort
				excludePorts[currentPort] = true // 次の割り当てで除外
				currentPort++
				break
			}
			currentPort++
		}

		// ポートが見つからなかった場合
		if _, found := result[service.Name]; !found {
			return result, fmt.Errorf("サービス %s のポート割り当てに失敗: 利用可能なポートがありません", service.Name)
		}
	}

	p.logger.Info(ctx, "サービス別ポート割り当て完了",
		types.Field{Key: "service_count", Value: len(result)},
		types.Field{Key: "allocations", Value: result})

	return result, nil
}
