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
- Terraform

## Quick start

```bash
# Docker Compose:
cp .env.example .env
docker compose up --build

# Local k3d cluster:
cd infra/terraform
cp terraform.tfvars.example terraform.tfvars
terraform init
terraform plan
terraform apply
terraform output port_forward
```

The API is available at `http://localhost:8080`, Prometheus at `http://localhost:9090`. 
