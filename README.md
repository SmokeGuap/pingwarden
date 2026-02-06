# Pingwarden

HTTP uptime monitoring service: periodically probes configured URLs, stores results, tracks state changes (up/down), and sends alerts to Telegram and/or a webhook.

## Tech stack

**Language and runtime**

- Go 1.25, standard library (`net/http`, `log/slog`, `context`)

**Data**

- PostgreSQL
- pgx/v5 + pgxpool
- goose
- Redis

**Observability**

- Prometheus

**Infrastructure**

- Docker 
- Docker Compose
- Helm chart for Kubernetes
- Ansible

## Quick start

```bash
cp .env.example .env
docker compose up --build
```

The API is available at `http://localhost:8080`, Prometheus at `http://localhost:9090`. 
