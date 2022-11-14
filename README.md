# ably-protocol-exercise

The protocol spec and implementation for streaming numbers to a client.

## Requirements

- Go >= 1.19

## Preparation

Copy `.env.client.example` to `.env.client` and `.env.server.example` to `.env.server`.

Modify any of the configuration if necessary.

## Build

Make sure your starting point in your terminal is the project root directory before
following the build instructions.

### Client

```bash
./scripts/build-client.sh
```

### Server

```bash
./scripts/build-server.sh
```

## Run

### Server

```bash
./bin/server --port 3049
```

The port can be any unused port that you would like to run the server.

### Client

```bash
./bin/client --server-host localhost --server-port 3049
```

With sequence count:

```bash
./bin/client --server-host localhost --server-port 3049 --sequence-count 200
```

The port must be the same port the server is running on.

## Testing

```bash
 go test ./...
```

Verbose (exposes stdout from code under test):

```bash
 go test ./... -v
```

Individual tests:

```bash
cd pkg/server
go test -timeout 30s -run ^Test_server_handles_concurrent_clients # for example, you can pick any of the test function names.
```

Tests that span the server and client are found in `pkg/server/server_test.go`.

## Debugging

Set `LOG_LEVEL` env var to `debug` in `.env.client` and `.env.server` to see debug logs.

## Further Documentation

- [Protocol Specification](/PROTOCOL.md)
- [Configuration](/CONFIG.md)
