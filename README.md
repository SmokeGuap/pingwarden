# Pingwarden

HTTP uptime monitoring service: periodically probes configured URLs, stores results, tracks state changes (up/down), and sends alerts to Telegram and/or a webhook.

## Tech stack

**Language and runtime**

- Go 1.27, standard library (`net/http`, `log/slog`, `context`) — no web framework

**Data**

- PostgreSQL 17 — checks and result history
- pgx/v5 + pgxpool
- goose
- Redis 8

**Observability**

- Prometheus

**Infrastructure**

- Docker 
- Docker Compose
- Helm chart for Kubernetes
- Ansible
- Terraform (Kubernetes namespace, secrets and Helm release)

## Quick start

```bash
# Docker Compose:
cp .env.example .env   # fill in notification channels if needed
docker compose up --build

# Local k3d cluster (context k3d-pingwarden):
cd infra/terraform
cp terraform.tfvars.example terraform.tfvars   # fill in notification channels if needed
terraform init                                # download providers, create lock file
terraform plan                                # review changes before applying
terraform apply                               # create namespace, secrets and Helm release
terraform output port_forward                 # prints the kubectl port-forward command
```

The API is available at `http://localhost:8080`, Prometheus at `http://localhost:9090`.