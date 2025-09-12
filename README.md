# Protohackers Solutions

This repository contains solutions for [Protohackers](https://protohackers.com/) problems implemented in Go.

## Structure

Each problem is organized in its own directory under `problems/`:

```
problems/
├── smoke-test/      # Problem 0: Smoke Test (Echo server)
│   ├── main.go
│   └── Dockerfile
├── prime-time/      # Problem 1: Prime Time (JSON primality testing)
│   ├── main.go
│   └── Dockerfile
└── means-to-end/    # Problem 2: Means to an End (Asset price tracking)
    ├── db/
    ├── sql/
    ├── main.go
    └── Dockerfile
```

## Building and Running

Each problem can be built and run independently:

### Using Docker

```bash
# Build and run smoke-test
cd problems/smoke-test
docker build -t smoke-test .
docker run -p 8000:8000 smoke-test

# Build and run prime-time
cd problems/prime-time
docker build -t prime-time .
docker run -p 8000:8000 prime-time

# Build and run means-to-end
cd problems/means-to-end
docker build -t means-to-end .
docker run -p 8000:8000 means-to-end
```

### Using Go directly

```bash
# Run smoke-test
cd problems/smoke-test
go run main.go

# Run prime-time
cd problems/prime-time
go run main.go

# Run means-to-end
cd problems/means-to-end
go run main.go
```

## Problems

- **Problem 0 - Smoke Test**: Echo server that accepts TCP connections and echoes back all received data
- **Problem 1 - Prime Time**: JSON-based TCP server that tests if numbers are prime
- **Problem 2 - Means to an End**: Asset price tracking server that stores timestamped prices and calculates means over time ranges