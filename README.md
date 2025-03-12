# inventory-api

Real-time inventory management API

## Tech Stack
- **Language**: go
- **Team**: supply-chain
- **Platform**: Walmart Global K8s

## Quick Start
```bash
docker build -t inventory-api:latest .
docker run -p 8080:8080 inventory-api:latest
curl http://localhost:8080/health
```

## API Endpoints
| Method | Path | Description |
|--------|------|-------------|
| GET | /health | Health check |
| GET | /ready | Readiness probe |
| GET | /metrics | Prometheus metrics |
## Inventory API Reference
