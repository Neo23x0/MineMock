package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

func main() {
	// Common miner flags (XMRig-style)
	pool := flag.String("o", "", "pool address (e.g., pool.minexmr.com:4444)")
	user := flag.String("u", "", "username/wallet address")
	pass := flag.String("p", "x", "password")
	threads := flag.Int("t", runtime.NumCPU(), "number of threads")
	donate := flag.Int("donate-level", 1, "donate level (simulated)")
	background := flag.Bool("B", false, "run in background (simulated)")
	
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

	if *pool == "" {
		fmt.Fprintln(os.Stderr, "pool address (-o) is required")
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
		log.Printf("NOTE: This is a simulation tool. No actual mining will occur.")
	}

	// Simulate pool connection
	conn := simulatePoolConnection(cleanPool, *verbose)
	if conn != nil {
		defer conn.Close()
	}

	// Start CPU load simulation
	var wg sync.WaitGroup
	stopChan := make(chan bool)

	for i := 0; i < *threads; i++ {
		wg.Add(1)
		go simulateWorker(i, *cpuLoad, &wg, stopChan, *verbose)
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
