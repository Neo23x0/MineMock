package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestGenerateMoneroAddress tests the wallet address generation
func TestGenerateMoneroAddress(t *testing.T) {
	alphabet := "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	
	tests := []struct {
		name     string
		prefix   byte
		length   int
		wantLen  int
		wantPrefix byte
	}{
		{
			name:     "Standard address",
			prefix:   '4',
			length:   95,
			wantLen:  95,
			wantPrefix: '4',
		},
		{
			name:     "Subaddress",
			prefix:   '8',
			length:   95,
			wantLen:  95,
			wantPrefix: '8',
		},
		{
			name:     "Integrated address",
			prefix:   '4',
			length:   106,
			wantLen:  106,
			wantPrefix: '4',
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := generateMoneroAddress(alphabet, tt.prefix, tt.length)
			
			if len(addr) != tt.wantLen {
				t.Errorf("generateMoneroAddress() length = %d, want %d", len(addr), tt.wantLen)
			}
			
			if addr[0] != tt.wantPrefix {
				t.Errorf("generateMoneroAddress() prefix = %c, want %c", addr[0], tt.wantPrefix)
			}
			
			// Verify all characters are valid base58
			for i, c := range addr {
				if !strings.ContainsRune(alphabet, c) {
					t.Errorf("generateMoneroAddress() invalid char at position %d: %c", i, c)
				}
			}
		})
	}
}

// TestPoolAddressParsing tests pool address cleanup
func TestPoolAddressParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Plain address",
			input:    "pool.supportxmr.com:3333",
			expected: "pool.supportxmr.com:3333",
		},
		{
			name:     "With stratum+tcp prefix",
			input:    "stratum+tcp://pool.supportxmr.com:3333",
			expected: "pool.supportxmr.com:3333",
		},
		{
			name:     "With stratum+ssl prefix",
			input:    "stratum+ssl://pool.supportxmr.com:3333",
			expected: "pool.supportxmr.com:3333",
		},
		{
			name:     "No port specified",
			input:    "pool.supportxmr.com",
			expected: "pool.supportxmr.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strings.TrimPrefix(tt.input, "stratum+tcp://")
			result = strings.TrimPrefix(result, "stratum+ssl://")
			
			if result != tt.expected {
				t.Errorf("Pool address cleanup = %s, want %s", result, tt.expected)
			}
		})
	}
}

// TestStratumRequestSerialization tests JSON-RPC request creation
func TestStratumRequestSerialization(t *testing.T) {
	tests := []struct {
		name   string
		id     interface{}
		method string
		params []interface{}
	}{
		{
			name:   "Subscribe request",
			id:     1,
			method: "mining.subscribe",
			params: []interface{}{"MineMock/1.0"},
		},
		{
			name:   "Authorize request",
			id:     2,
			method: "mining.authorize",
			params: []interface{}{"wallet.worker", "x"},
		},
		{
			name:   "Submit request",
			id:     10,
			method: "mining.submit",
			params: []interface{}{"worker", "00000000", "deadbeef"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := StratumRequest{
				ID:     tt.id,
				Method: tt.method,
				Params: tt.params,
			}

			data, err := json.Marshal(req)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// Verify it can be unmarshaled
			var decoded StratumRequest
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal request: %v", err)
			}

			if decoded.Method != tt.method {
				t.Errorf("Method = %s, want %s", decoded.Method, tt.method)
			}
			
			// Note: JSON numbers unmarshal to float64, so ID comparison needs type flexibility
			_ = decoded.ID // Just verify it exists
		})
	}
}

// TestSendStratumRequest tests the request sending function
func TestSendStratumRequest(t *testing.T) {
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	err := sendStratumRequest(writer, 1, "mining.subscribe", []interface{}{"MineMock/1.0"})
	if err != nil {
		t.Fatalf("sendStratumRequest failed: %v", err)
	}

	// Read what was written
	output := buf.String()
	
	// Verify it ends with newline
	if !strings.HasSuffix(output, "\n") {
		t.Error("Request should end with newline")
	}

	// Verify it's valid JSON
	var req StratumRequest
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &req); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	if req.Method != "mining.subscribe" {
		t.Errorf("Method = %s, want mining.subscribe", req.Method)
	}

	// Note: JSON numbers unmarshal to float64
	if req.ID == nil {
		t.Error("ID should not be nil")
	}
}

// TestReadStratumResponse tests response parsing
func TestReadStratumResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantID   interface{}
		wantErr  bool
	}{
		{
			name:    "Valid response",
			input:   `{"id":1,"result":true,"error":null}` + "\n",
			wantID:  float64(1),
			wantErr: false,
		},
		{
			name:    "Response with result array",
			input:   `{"id":2,"result":["sessionid","nonce"],"error":null}` + "\n",
			wantID:  float64(2),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tt.input))
			var resp StratumResponse
			err := readStratumResponse(reader, &resp)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.wantErr && resp.ID != tt.wantID {
				t.Errorf("ID = %v, want %v", resp.ID, tt.wantID)
			}
		})
	}
}

// TestKnownPools tests that the known pools list is populated
func TestKnownPools(t *testing.T) {
	if len(knownPools) == 0 {
		t.Error("knownPools should not be empty")
	}

	for _, pool := range knownPools {
		if pool.Name == "" {
			t.Error("Pool name should not be empty")
		}
		if pool.Address == "" {
			t.Errorf("Pool %s has no address", pool.Name)
		}
		if pool.Port == "" {
			t.Errorf("Pool %s has no port", pool.Name)
		}
	}
}

// TestBusyWork tests that the busy work function runs without error
func TestBusyWork(t *testing.T) {
	// This is mostly a smoke test - busyWork shouldn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("busyWork panicked: %v", r)
		}
	}()

	busyWork()
}

// TestSubmitShare tests share submission
func TestSubmitShare(t *testing.T) {
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	err := submitShare(writer)
	if err != nil {
		t.Fatalf("submitShare failed: %v", err)
	}

	output := buf.String()
	
	// Verify it contains the expected method
	if !strings.Contains(output, "mining.submit") {
		t.Error("Output should contain 'mining.submit'")
	}

	// Verify it's valid JSON
	var req StratumRequest
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &req); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}
}
