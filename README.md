# Protohackers Solutions

This repository contains solutions for [Protohackers](https://protohackers.com/) problems implemented in Go.

## About Protohackers

Protohackers is a series of network programming challenges that test your ability to build robust TCP/UDP servers following specific protocols. Each problem increases in complexity, covering various networking concepts and protocol implementations.

## Project Structure

```
problems/
├── line-reversal/           # Problem 7: Custom LRCP protocol implementation
├── speed-daemon/            # Problem 6: Speed camera and ticketing system
├── mob-in-middle/           # Problem 5: TCP proxy with address rewriting
├── unusal-database-program/ # Problem 4: UDP key-value store
├── budget-chat/             # Problem 3: Multi-user chat room
├── means-to-end/            # Problem 2: Asset price tracking with binary protocol
├── prime-time/              # Problem 1: JSON primality testing
└── smoke-test/              # Problem 0: Echo server

tools/
└── lrcp-client/             # LRCP protocol testing client
```

## Problems Solved

### Problem 7: Line Reversal
**Protocol:** LRCP (Line Reversal Control Protocol)

Implementation of a custom reliable protocol (LRCP) over UDP that reverses lines of text.

**Key Features:**
- Custom protocol implementation with:
  - Message acknowledgments
  - Retransmission on timeout
  - Session management
  - Packet ordering
- Reverses each line of received text
- Custom protocol package with Listener and Connection types

**Implementation:**
- Main: problems/line-reversal/main.go:10-78
- Protocol: problems/line-reversal/protocol/

### Problem 6: Speed Daemon
**Protocol:** Binary TCP with Multiple Message Types

A sophisticated speed camera and ticketing system that processes vehicle observations and issues speeding tickets.

**Key Features:**
- Camera client protocol for plate observations
- Dispatcher client protocol for receiving tickets
- Heartbeat mechanism for keeping connections alive
- Speed limit enforcement with ticket generation
- SQLite database for observations, roads, and tickets
- Prevents duplicate tickets for the same day
- Background processing of observations and unprocessed tickets

**Implementation:** problems/speed-daemon/main.go:1-544

### Problem 5: Mob in the Middle
**Protocol:** TCP Proxy

A man-in-the-middle proxy that intercepts chat messages and rewrites Boguscoin addresses to redirect payments.

**Key Features:**
- Proxies connections to `chat.protohackers.com:16963`
- Rewrites Boguscoin addresses (26-35 alphanumeric chars starting with '7')
- Bidirectional message forwarding
- Concurrent goroutines for reading both directions

**Implementation:** problems/mob-in-middle/main.go:12-140

### Problem 4: Unusual Database Program
**Protocol:** UDP Key-Value Store

A UDP-based key-value database that supports insert and retrieve operations via simple string protocol.

**Key Features:**
- UDP packet-based communication
- Insert: `key=value` format
- Retrieve: send key, receive `key=value`
- Special immutable `version` key
- In-memory hash map storage

**Implementation:** problems/unusal-database-program/main.go:10-91

### Problem 3: Budget Chat
**Protocol:** TCP Chat Server

A multi-user chat room server where users can join, send messages, and see who else is in the room.

**Key Features:**
- Username validation (alphanumeric only)
- Broadcasts join/leave notifications
- Lists current users on join
- Concurrent message handling with mutex locks
- Per-user goroutines for message handling

**Implementation:**
- Main: problems/budget-chat/main.go:8-53
- Room management: problems/budget-chat/room.go:10-96
- User handling: problems/budget-chat/user.go:14-102

### Problem 2: Means to an End
**Protocol:** Binary TCP

An asset price tracking server using a custom binary protocol. Clients can insert timestamped prices and query for the mean price over time ranges.

**Key Features:**
- Binary protocol with 9-byte messages
- Insert command: Store timestamp and price pairs
- Query command: Calculate mean price over time range
- SQLite database for price storage
- Connection-specific asset tracking

**Implementation:** problems/means-to-end/main.go:20-216

### Problem 1: Prime Time
**Protocol:** JSON over TCP

A TCP server that receives JSON requests asking whether numbers are prime and responds with JSON answers. Handles malformed requests gracefully and supports floating-point number validation.

**Key Features:**
- JSON request/response protocol
- Prime number checking using the `fxtlabs/primes` library
- Validates that numbers are integers (no fractional parts)
- Error handling for malformed JSON

**Implementation:** problems/prime-time/main.go:46-137

### Problem 0: Smoke Test
**Protocol:** TCP Echo Server

A simple TCP server that echoes back whatever it receives. Great for testing basic TCP connection handling.

**Implementation:** problems/smoke-test/main.go:31-42

## Building and Running

Each problem can be built and run independently using Docker or Go directly.

### Using Docker

```bash
# General pattern
cd problems/<problem-name>
docker build -t <problem-name> .
docker run -p 8000:8000 <problem-name>

# Example: Run the chat server
cd problems/budget-chat
docker build -t budget-chat .
docker run -p 8000:8000 budget-chat
```

### Using Go Directly

```bash
# General pattern
cd problems/<problem-name>
go run main.go

# Example: Run the speed daemon
cd problems/speed-daemon
go run main.go
```

All servers listen on port `8000` by default.

## Technologies Used

- **Go** - Primary programming language
- **SQLite** - Database for problems requiring persistence (means-to-end, speed-daemon)
- **sqlc** - SQL code generation (see sqlc.json)
- **Docker** - Containerization for each solution

## Dependencies

Key Go packages used:
- `github.com/fxtlabs/primes` - Prime number checking
- `modernc.org/sqlite` - Pure Go SQLite implementation
- Standard library: `net`, `bufio`, `encoding/json`, `encoding/binary`

## Testing

Protohackers provides an automated testing system. Submit your server's address to the challenge page for validation.

For local testing of LRCP (Problem 7), use the included client:
```bash
cd tools/lrcp-client
go run main.go
```

## License

This is a personal learning project for the Protohackers challenges.