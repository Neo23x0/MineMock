package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// StratumRequest represents a Stratum protocol request
type StratumRequest struct {
	ID     interface{}   `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

// StratumResponse represents a Stratum protocol response
type StratumResponse struct {
	ID     interface{} `json:"id"`
	Result interface{} `json:"result"`
	Error  interface{} `json:"error,omitempty"`
}

// StratumNotification represents a job notification from the pool
type StratumNotification struct {
	ID     interface{}   `json:"id,omitempty"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

// Known mining pools for reference and quick selection
type PoolInfo struct {
	Name    string
	Address string
	Port    string
	Algo    string
}

var knownPools = []PoolInfo{
	{Name: "supportxmr", Address: "pool.supportxmr.com", Port: "3333", Algo: "RandomX"},
	{Name: "xmrpool", Address: "xmrpool.eu", Port: "3333", Algo: "RandomX"},
	{Name: "moneroocean", Address: "gulf.moneroocean.stream", Port: "10128", Algo: "Auto"},
	{Name: "nanopool", Address: "xmr-eu1.nanopool.org", Port: "10300", Algo: "RandomX"},
	{Name: "c3pool", Address: "xmr.c3pool.org", Port: "3333", Algo: "RandomX"},
	{Name: "minexmr", Address: "pool.minexmr.com", Port: "4444", Algo: "RandomX"},
	{Name: "hashvault", Address: "xmr.hashvault.pro", Port: "3333", Algo: "RandomX"},
	{Name: "herominers", Address: "xmr.herominers.com", Port: "10191", Algo: "RandomX"},
	{Name: "kryptex", Address: "xmr.kryptex.network", Port: "3333", Algo: "RandomX"},
	{Name: "unmineable", Address: "rx.unmineable.com", Port: "3333", Algo: "RandomX"},
}

var (
	shareCount  uint64
	jobCount    uint64
	stratumConn net.Conn
	stratumMu   sync.Mutex
)

// generateWalletAddress generates a realistic-looking Monero wallet address
// for testing purposes. These addresses are NOT valid and cannot receive funds.
func generateWalletAddress() {
	fmt.Println("MineMock Wallet Address Generator")
	fmt.Println("=================================")
	fmt.Println()
	fmt.Println("Generating test wallet addresses for detection testing.")
	fmt.Println("NOTE: These addresses are NOT valid and cannot receive funds.")
	fmt.Println()

	// Monero uses Base58 encoding with specific alphabet
	base58Chars := "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	
	// Generate standard address (starts with 4, 95 chars)
	fmt.Println("Standard Address (95 chars, starts with '4'):")
	standardAddr := generateMoneroAddress(base58Chars, '4', 95)
	fmt.Println(standardAddr)
	fmt.Println()
	
	// Generate subaddress (starts with 8, 95 chars)
	fmt.Println("Subaddress (95 chars, starts with '8'):")
	subAddr := generateMoneroAddress(base58Chars, '8', 95)
	fmt.Println(subAddr)
	fmt.Println()
	
	// Generate integrated address (starts with 4, 106 chars)
	fmt.Println("Integrated Address (106 chars, starts with '4'):")
	integratedAddr := generateMoneroAddress(base58Chars, '4', 106)
	fmt.Println(integratedAddr)
	fmt.Println()
	
	fmt.Println("Usage example:")
	fmt.Printf("  minemock -o pool.supportxmr.com:3333 -u %s -t 4 --stratum\n", standardAddr[:20]+"...")
}

// generateMoneroAddress creates a fake Monero-style address
func generateMoneroAddress(alphabet string, prefix byte, length int) string {
	addr := make([]byte, length)
	addr[0] = prefix
	
	// Fill rest with random base58 characters
	for i := 1; i < length; i++ {
		addr[i] = alphabet[rand.Intn(len(alphabet))]
	}
	
	return string(addr)
}

func main() {
	// Handle subcommands before flag parsing
	if len(os.Args) > 1 && os.Args[1] == "gen-address" {
		generateWalletAddress()
		os.Exit(0)
	}

	// Common miner flags (XMRig-style)
	pool := flag.String("o", "", "pool address (e.g., pool.supportxmr.com:3333)")
	user := flag.String("u", "", "username/wallet address")
	pass := flag.String("p", "x", "password")
	threads := flag.Int("t", runtime.NumCPU(), "number of threads")
	donate := flag.Int("donate-level", 1, "donate level (simulated)")
	background := flag.Bool("B", false, "run in background (simulated)")
	
	// Protocol simulation
	stratum := flag.Bool("stratum", false, "enable Stratum protocol simulation (JSON-RPC)")
	listPools := flag.Bool("list-pools", false, "list top mining pools and exit")
	
	// Additional flags for realism
	cpuLoad := flag.Int("cpu-load", 50, "simulated CPU load percentage (1-100)")
	duration := flag.Int("duration", 0, "run duration in seconds (0 = infinite)")
	verbose := flag.Bool("v", false, "verbose output")
	
	// Common XMRig config options
	flag.String("c", "", "config file (simulated)")
	flag.String("k", "", "keepalive (simulated)")
	flag.Bool("nicehash", false, "nicehash mode (simulated)")
	flag.Bool("tls", false, "TLS mode (simulated)")
	
	flag.Parse()

	// Handle list-pools flag
	if *listPools {
		fmt.Println("Top Mining Pools for Testing:")
		fmt.Println("==============================")
		for i, p := range knownPools {
			fmt.Printf("%2d. %-15s %-30s Port: %-5s Algo: %s\n", 
				i+1, p.Name, p.Address, p.Port, p.Algo)
		}
		fmt.Println("\nUsage example:")
		fmt.Printf("  minemock -o %s:%s -u YOUR_WALLET -t 4 --stratum\n", 
			knownPools[0].Address, knownPools[0].Port)
		os.Exit(0)
	}

	if *pool == "" {
		fmt.Fprintln(os.Stderr, "pool address (-o) is required (use -list-pools to see options)")
		flag.Usage()
		os.Exit(1)
	}

	if *user == "" {
		fmt.Fprintln(os.Stderr, "username/wallet (-u) is required")
		flag.Usage()
		os.Exit(1)
	}

	// Clean up pool address (remove stratum+tcp:// prefix if present)
	cleanPool := strings.TrimPrefix(*pool, "stratum+tcp://")
	cleanPool = strings.TrimPrefix(cleanPool, "stratum+ssl://")

	if *verbose {
		log.Printf("MineMock starting...")
		log.Printf("Pool: %s", cleanPool)
		log.Printf("User: %s", *user)
		log.Printf("Threads: %d", *threads)
		log.Printf("CPU Load: %d%%", *cpuLoad)
		log.Printf("Stratum Protocol: %v", *stratum)
		log.Printf("NOTE: This is a simulation tool. No actual mining will occur.")
	}

	// Handle Stratum protocol or simple TCP connection
	if *stratum {
		go runStratumClient(cleanPool, *user, *pass, *verbose)
	} else {
		// Simple TCP connection for network artifact only
		conn := simulatePoolConnection(cleanPool, *verbose)
		if conn != nil {
			defer conn.Close()
		}
	}

	// Start CPU load simulation
	var wg sync.WaitGroup
	stopChan := make(chan bool)

	for i := 0; i < *threads; i++ {
		wg.Add(1)
		go simulateWorker(i, *cpuLoad, &wg, stopChan, *verbose)
	}

	// Start periodic stats reporter
	if *stratum {
		go statsReporter(*verbose)
	}

	// Handle duration
	if *duration > 0 {
		if *verbose {
			log.Printf("Running for %d seconds...", *duration)
		}
		time.Sleep(time.Duration(*duration) * time.Second)
		close(stopChan)
	} else {
		// Run indefinitely until interrupted
		if *verbose {
			log.Printf("Running indefinitely (Ctrl+C to stop)...")
		}
		// Wait for interrupt signal
		select {}
	}

	wg.Wait()
	
	if *verbose {
		log.Printf("MineMock stopped.")
	}
}

// runStratumClient connects to a pool and speaks the Stratum protocol
func runStratumClient(poolAddr, user, pass string, verbose bool) {
	// Add default port if not specified
	if !strings.Contains(poolAddr, ":") {
		poolAddr = poolAddr + ":3333"
	}

	if verbose {
		log.Printf("[Stratum] Connecting to pool: %s", poolAddr)
	}

	// Establish TCP connection
	conn, err := net.DialTimeout("tcp", poolAddr, 10*time.Second)
	if err != nil {
		if verbose {
			log.Printf("[Stratum] Pool connection failed: %v", err)
		}
		return
	}
	defer conn.Close()

	stratumMu.Lock()
	stratumConn = conn
	stratumMu.Unlock()

	if verbose {
		log.Printf("[Stratum] Connected to pool")
	}

	// Create buffered reader/writer
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// Send mining.subscribe
	if err := sendStratumRequest(writer, 1, "mining.subscribe", []interface{}{"MineMock/1.0"}); err != nil {
		if verbose {
			log.Printf("[Stratum] Failed to send subscribe: %v", err)
		}
		return
	}

	// Read subscribe response
	var subResp StratumResponse
	if err := readStratumResponse(reader, &subResp); err != nil {
		if verbose {
			log.Printf("[Stratum] Failed to read subscribe response: %v", err)
		}
		return
	}

	if verbose {
		log.Printf("[Stratum] Subscribed to pool")
	}

	// Send mining.authorize
	worker := user
	if !strings.Contains(user, ".") {
		worker = user + ".minemock"
	}
	if err := sendStratumRequest(writer, 2, "mining.authorize", []interface{}{worker, pass}); err != nil {
		if verbose {
			log.Printf("[Stratum] Failed to send authorize: %v", err)
		}
		return
	}

	// Read authorize response
	var authResp StratumResponse
	if err := readStratumResponse(reader, &authResp); err != nil {
		if verbose {
			log.Printf("[Stratum] Failed to read authorize response: %v", err)
		}
		return
	}

	if verbose {
		log.Printf("[Stratum] Authorized as worker: %s", worker)
	}

	// Main loop: read notifications and send periodic submits
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Simulate submitting a share
			if err := submitShare(writer); err != nil {
				if verbose {
					log.Printf("[Stratum] Failed to submit share: %v", err)
				}
			} else {
				atomic.AddUint64(&shareCount, 1)
				if verbose {
					log.Printf("[Stratum] Share submitted")
				}
			}
		default:
			// Try to read with timeout
			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			var notif StratumNotification
			if err := readStratumNotification(reader, &notif); err == nil {
				if notif.Method == "mining.notify" {
					atomic.AddUint64(&jobCount, 1)
					if verbose {
						log.Printf("[Stratum] New job received")
					}
				} else if notif.Method == "mining.set_difficulty" {
					if verbose {
						log.Printf("[Stratum] Difficulty adjusted")
					}
				}
			}
		}
	}
}

// sendStratumRequest sends a JSON-RPC request to the pool
func sendStratumRequest(writer *bufio.Writer, id interface{}, method string, params []interface{}) error {
	req := StratumRequest{
		ID:     id,
		Method: method,
		Params: params,
	}
	
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	
	_, err = writer.WriteString(string(data) + "\n")
	if err != nil {
		return err
	}
	
	return writer.Flush()
}

// readStratumResponse reads a JSON-RPC response from the pool
func readStratumResponse(reader *bufio.Reader, resp *StratumResponse) error {
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(line), resp)
}

// readStratumNotification reads a JSON-RPC notification from the pool
func readStratumNotification(reader *bufio.Reader, notif *StratumNotification) error {
	line, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(line), notif)
}

// submitShare simulates submitting a mining share
func submitShare(writer *bufio.Writer) error {
	// Generate fake nonce and hash
	nonce := fmt.Sprintf("%08x", rand.Uint32())
	hash := fmt.Sprintf("%064x", rand.Uint64())
	
	return sendStratumRequest(writer, rand.Intn(1000)+10, "mining.submit", []interface{}{
		"minemock",
		nonce,
		hash,
	})
}

// statsReporter periodically logs statistics
func statsReporter(verbose bool) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		shares := atomic.LoadUint64(&shareCount)
		jobs := atomic.LoadUint64(&jobCount)
		if verbose {
			log.Printf("[Stats] Jobs received: %d, Shares submitted: %d", jobs, shares)
		}
	}
}

// simulatePoolConnection attempts to connect to the pool address
// In real mining, this would use Stratum protocol. Here we just open
// a TCP connection to simulate the network artifact.
func simulatePoolConnection(poolAddr string, verbose bool) net.Conn {
	// Add default port if not specified
	if !strings.Contains(poolAddr, ":") {
		poolAddr = poolAddr + ":3333"
	}

	if verbose {
		log.Printf("Connecting to pool: %s", poolAddr)
	}

	// Try to establish connection (this creates the network artifact)
	conn, err := net.DialTimeout("tcp", poolAddr, 10*time.Second)
	if err != nil {
		if verbose {
			log.Printf("Pool connection failed (expected in simulation): %v", err)
		}
		// Return nil - we still want to simulate CPU load even if pool is unreachable
		return nil
	}

	if verbose {
		log.Printf("Connected to pool (simulated - no actual mining protocol)")
	}

	return conn
}

// simulateWorker simulates CPU work by consuming cycles
// The load parameter controls what percentage of time is spent "working"
func simulateWorker(id int, load int, wg *sync.WaitGroup, stopChan chan bool, verbose bool) {
	defer wg.Done()

	if load < 1 {
		load = 1
	}
	if load > 100 {
		load = 100
	}

	workDuration := time.Duration(load) * time.Millisecond
	sleepDuration := time.Duration(100-load) * time.Millisecond

	if verbose {
		log.Printf("Worker %d started (load: %d%%)", id, load)
	}

	for {
		select {
		case <-stopChan:
			if verbose {
				log.Printf("Worker %d stopping", id)
			}
			return
		default:
			// Simulate work by burning CPU cycles
			busyWork()
			time.Sleep(workDuration)
			
			// Rest period to control overall load
			if sleepDuration > 0 {
				time.Sleep(sleepDuration)
			}
		}
	}
}

// busyWork performs CPU-intensive calculations to simulate mining
func busyWork() {
	// Simple busy work - calculate some hashes (not actual mining hashes)
	// This consumes CPU without doing anything useful
	var result uint64 = 1
	for i := 0; i < 100000; i++ {
		result = result*uint64(i+1) + uint64(i)
		if result > 1<<60 {
			result = 1
		}
	}
	// Use the result to prevent optimization
	_ = result
}
