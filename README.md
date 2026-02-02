![MineMock logo](minemock.png)

# MineMock

A harmless cryptocurrency miner simulator for detection testing. Generates realistic mining-like artifacts without consuming resources or connecting to actual pools.

## What is MineMock

MineMock is a small Golang tool that imitates the behavior patterns of cryptocurrency mining software without performing any actual mining. It simulates CPU load patterns, creates network connections to common mining pool addresses, and mimics command-line arguments used by popular miners (XMRig, CGMiner, etc.) — all without wasting electricity, generating profit, or participating in botnet activity.

## Features

- **Realistic CPU patterns** — Generates configurable CPU load spikes that resemble mining algorithms
- **Pool connection simulation** — Opens TCP connections to well-known mining pool domains/IPs
- **Stratum protocol simulation** — Optional full Stratum protocol handshake (`--stratum` flag)
  - `mining.subscribe` — Subscribe to pool
  - `mining.authorize` — Authenticate worker
  - `mining.notify` — Receive job notifications
  - `mining.submit` — Submit simulated shares
  - `mining.set_difficulty` — Handle difficulty adjustments
- **Command-line mimicry** — Accepts common miner flags like `-o` (pool), `-u` (wallet/user), `-p` (password), `-t` (threads)
- **Built-in pool list** — Reference list of top 10 mining pools (`--list-pools`)
- **No actual mining** — No hash calculations, no wallet addresses validated, no real shares submitted

## Purpose

Detection engineering for crypto-mining threats requires:

- Realistic process behavior patterns
- Suspicious network connection artifacts (to pools like `pool.supportxmr.com`, `xmrpool.eu`)
- High CPU usage signatures
- Command-line patterns with wallet addresses
- **Stratum protocol detection** — Network signatures for JSON-RPC mining protocol

Using real mining software in enterprise environments risks actual resource abuse and security policy violations. MineMock provides the detection artifacts without the risk.

## Build

MineMock is pure Go and can be built or cross-compiled easily.  
Requires **Go 1.20+**.

### Build for the current platform

```bash
go build -o minemock minemock.go
```

### Cross-compile for Windows

```bash
GOOS=windows GOARCH=amd64 go build -o minemock_windows_amd64.exe minemock.go
GOOS=windows GOARCH=arm64 go build -o minemock_windows_arm64.exe minemock.go
```

### Cross-compile for macOS

```bash
GOOS=darwin GOARCH=amd64 go build -o minemock_darwin_amd64 minemock.go
GOOS=darwin GOARCH=arm64 go build -o minemock_darwin_arm64 minemock.go
```

### Unified build script (`build.sh`)

The repository includes a convenience script that builds all supported platforms in one run.

Make executable:

```bash
chmod +x build.sh
```

Run:

```bash
./build.sh
```

## Usage Examples

### List top mining pools

```bash
minemock --list-pools
```

### Basic simulation (TCP connection only)

```bash
minemock -o pool.supportxmr.com:3333 -u 44abcd...wallet -p x -t 4
```

### Full Stratum protocol simulation

```bash
minemock -o pool.supportxmr.com:3333 -u 44abcd...wallet -p x -t 4 --stratum -v
```

This enables:
- JSON-RPC Stratum handshake
- Periodic share submissions (simulated)
- Job notification handling
- Keepalive traffic

### Custom CPU load pattern

```bash
minemock -o xmrpool.eu:3333 -u worker@evil.com -t 8 --cpu-load=75 --duration=300 --stratum
```

### Background mode (common in actual attacks)

```bash
nohup ./minemock -o pool.supportxmr.com:3333 -u 44... -p x -t 4 --stratum -B > /dev/null 2>&1 &
```

## Network Detection Testing

### Stratum Protocol Signatures

When using `--stratum`, MineMock generates realistic Stratum protocol traffic:

**Outgoing (Client → Pool):**
```json
{"id":1,"method":"mining.subscribe","params":["MineMock/1.0"]}
{"id":2,"method":"mining.authorize","params":["WALLET.worker","x"]}
{"id":10,"method":"mining.submit","params":["worker","nonce","hash"]}
```

**Incoming (Pool → Client):**
```json
{"id":null,"method":"mining.notify","params":[...]}
{"id":null,"method":"mining.set_difficulty","params":[...]}
```

These patterns can be used to test:
- IDS/IPS rules for Stratum protocol detection
- Network traffic analysis for mining pool connections
- Behavioral analytics for periodic share submission patterns

## Relevant Sigma Rules

MineMock is designed to mimic common cryptocurrency miner command-line patterns and network behaviors without performing harmful resource consumption.

When used in detection pipelines, it can help validate rules that look for:

- Suspicious miner process names
- Connections to known mining pools
- High CPU usage patterns
- Command-line patterns with wallet addresses
- **Stratum protocol over non-standard ports**

## Top Mining Pools Reference

MineMock includes a built-in list of commonly used mining pools:

| # | Name | Address | Port | Algorithm |
|---|------|---------|------|-----------|
| 1 | supportxmr | pool.supportxmr.com | 3333 | RandomX |
| 2 | xmrpool | xmrpool.eu | 3333 | RandomX |
| 3 | moneroocean | gulf.moneroocean.stream | 10128 | Auto |
| 4 | nanopool | xmr-eu1.nanopool.org | 10300 | RandomX |
| 5 | c3pool | xmr.c3pool.org | 3333 | RandomX |
| 6 | minexmr | pool.minexmr.com | 4444 | RandomX |
| 7 | hashvault | xmr.hashvault.pro | 3333 | RandomX |
| 8 | herominers | xmr.herominers.com | 10191 | RandomX |
| 9 | kryptex | xmr.kryptex.network | 3333 | RandomX |
| 10 | unmineable | rx.unmineable.com | 3333 | RandomX |

## License

MIT
