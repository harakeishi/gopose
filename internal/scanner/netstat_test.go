package scanner

import (
	"sort"
	"testing"

	"github.com/harakeishi/gopose/internal/testutil"
)

func TestParseNetstatOutput(t *testing.T) {
	testLogger := testutil.NewTestLogger()

	tests := []struct {
		name          string
		input         string
		expectedPorts []int
	}{
		{
			name:          "empty output returns empty ports",
			input:         "",
			expectedPorts: []int{},
		},
		{
			name: "single tcp LISTEN line extracts correct port",
			input: `Active Internet connections (including servers)
Proto Recv-Q Send-Q  Local Address          Foreign Address        (state)
tcp4       0      0  127.0.0.1.8080         *.*                    LISTEN`,
			expectedPorts: []int{8080},
		},
		{
			name: "multiple LISTEN lines with duplicate are deduplicated",
			input: `Active Internet connections (including servers)
Proto Recv-Q Send-Q  Local Address          Foreign Address        (state)
tcp4       0      0  127.0.0.1.8080         *.*                    LISTEN
tcp4       0      0  *.8080                 *.*                    LISTEN
tcp4       0      0  127.0.0.1.3000         *.*                    LISTEN`,
			expectedPorts: []int{3000, 8080},
		},
		{
			name: "ignore non-LISTEN lines like ESTABLISHED and TIME_WAIT",
			input: `Active Internet connections (including servers)
Proto Recv-Q Send-Q  Local Address          Foreign Address        (state)
tcp4       0      0  127.0.0.1.8080         *.*                    LISTEN
tcp4       0      0  192.168.1.5.52341      93.184.216.34.443      ESTABLISHED
tcp4       0      0  192.168.1.5.52342      93.184.216.34.80       TIME_WAIT`,
			expectedPorts: []int{8080},
		},
		{
			name: "tcp46 format is parsed correctly",
			input: `Active Internet connections (including servers)
Proto Recv-Q Send-Q  Local Address          Foreign Address        (state)
tcp46      0      0  *.9090                 *.*                    LISTEN`,
			expectedPorts: []int{9090},
		},
		{
			name: "udp LISTEN line is parsed correctly",
			input: `Active Internet connections (including servers)
Proto Recv-Q Send-Q  Local Address          Foreign Address        (state)
udp4       0      0  127.0.0.1.5353         *.*                    LISTEN`,
			expectedPorts: []int{5353},
		},
		{
			name: "no LISTEN lines at all returns empty ports",
			input: `Active Internet connections (including servers)
Proto Recv-Q Send-Q  Local Address          Foreign Address        (state)
tcp4       0      0  192.168.1.5.52341      93.184.216.34.443      ESTABLISHED
tcp4       0      0  192.168.1.5.52342      93.184.216.34.80       TIME_WAIT
tcp4       0      0  192.168.1.5.52343      93.184.216.34.80       CLOSE_WAIT`,
			expectedPorts: []int{},
		},
		{
			name: "mixed LISTEN and non-LISTEN returns only LISTEN ports",
			input: `Active Internet connections (including servers)
Proto Recv-Q Send-Q  Local Address          Foreign Address        (state)
tcp4       0      0  127.0.0.1.5432         *.*                    LISTEN
tcp4       0      0  192.168.1.5.52341      93.184.216.34.443      ESTABLISHED
tcp46      0      0  *.3306                 *.*                    LISTEN
tcp4       0      0  192.168.1.5.52342      93.184.216.34.80       TIME_WAIT
udp4       0      0  127.0.0.1.8125         *.*                    LISTEN
tcp4       0      0  192.168.1.5.52343      93.184.216.34.80       CLOSE_WAIT`,
			expectedPorts: []int{3306, 5432, 8125},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewNetstatPortDetector(testLogger)
			ports, err := detector.parseNetstatOutput(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			sort.Ints(ports)

			if len(tt.expectedPorts) == 0 && len(ports) == 0 {
				return
			}

			if len(ports) != len(tt.expectedPorts) {
				t.Fatalf("expected %d ports %v, got %d ports %v",
					len(tt.expectedPorts), tt.expectedPorts,
					len(ports), ports)
			}

			for i, expected := range tt.expectedPorts {
				if ports[i] != expected {
					t.Errorf("port[%d]: expected %d, got %d", i, expected, ports[i])
				}
			}
		})
	}
}
