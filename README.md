# Data Agent

Lightweight system metrics collector (CPU, RAM, Disk, Network) with RabbitMQ, PostgreSQL, and gRPC API.

## 🚀 Quick Start

1. **Configure**: `cp .env.example .env` (edit if needed).
2. **Infrastructure**: `make up` (starts DB, RabbitMQ, API, Ingestor).
3. **Agent**: `make run-agent` (starts collecting metrics).

## 🛠 Development

All tasks are managed via `Makefile`:

- `make install-deps` — Install gRPC plugins.
- `make gen`          — Regenerate gRPC code from `.proto`.
- `make test`         — Run all tests.
- `make clean`        — Remove binaries and garbage.

## 📡 gRPC API

- **HostService**: `ListHosts`, `GetHost`
- **MetricService**: `ListMetrics`, `GetLatestMetrics` (returns structured data).

Example using `grpcurl`:
```bash
grpcurl -plaintext localhost:50051 data_agent.MetricService/ListMetrics
```

## License
MIT