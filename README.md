# Data Agent

Lightweight system metrics collector (CPU, RAM, Disk, Network) with RabbitMQ, PostgreSQL, and gRPC API.

## Quick Start

1. **Configure**: `cp .env.example .env` (edit if needed).
2. **Infrastructure**: `make up` (starts DB, RabbitMQ, API, Ingestor).
3. **Agent**: `make run-agent` (starts collecting metrics).

## Development

All tasks are managed via Makefile:

- `make install-deps` — Install gRPC plugins.
- `make gen`          — Regenerate gRPC code from .proto.
- `make build`        — Build the agent binary.
- `make test`         — Run all tests.
- `make clean`        — Remove build artifacts.

## Raspberry Pi Deployment

To run the agent on a Raspberry Pi Zero 2W, cross-compile for ARM64:
```bash
GOOS=linux GOARCH=arm64 go build -o agent_pi cmd/agent/main.go
```

## gRPC API

- **HostService**: ListHosts, GetHost
- **MetricService**: ListMetrics, GetLatestMetrics (structured data).

Example using grpcurl:
```bash
grpcurl -plaintext localhost:50051 data_agent.MetricService/ListMetrics
```

## License
This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
