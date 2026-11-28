# Pingwarden

HTTP uptime monitoring service: periodically probes configured URLs, stores results, tracks state changes (up/down), and sends alerts to Telegram and/or a webhook.

## Tech stack

**Language and runtime**

- Go 1.25, standard library (`net/http`, `log/slog`, `context`)
- Services: `api`, `scheduler`, `worker`, `notifier`, `migrate`

**Data**

- PostgreSQL
- pgx/v5 + pgxpool
- goose
- Redis

**Security**

- Bearer-token auth for the API
- SSRF guard: target IPs are checked after DNS resolution (dialer level)

**Observability**

- Prometheus
- Alertmanager
- Grafana
- Loki
- Vector

**Infrastructure**

- Docker, Docker Compose
- Kubernetes: k3s (server, via Ansible), k3d (local cluster)
- Helm (`deploy/helm/pingwarden`, `deploy/helm/monitoring`)
- Ansible
- Terraform

## Quick start

### Docker Compose

```bash
cp .env.example .env
docker compose up --build
```

- API: `http://localhost:8080`
- Prometheus: `http://localhost:9090`

### Local Kubernetes (k3d + Terraform)

```bash
k3d cluster create pingwarden -p "8081:80@loadbalancer"
docker build -t pingwarden:dev .
k3d image import pingwarden:dev -c pingwarden

cd infra/terraform
cp terraform.tfvars.example terraform.tfvars
terraform init
terraform plan
terraform apply
terraform output port_forward
terraform output monitoring_port_forwards
export TOKEN=$(terraform output -raw api_token)
terraform output -raw grafana_password
```

- API: `http://localhost:8080`
- Grafana: `http://localhost:3000`
- Prometheus: `http://localhost:9091`
- Alertmanager: `http://localhost:9093`

### Server with k3s (Ansible)

Ansible prepares a clean Ubuntu 24 server (non-root user, SSH keys only, ufw, fail2ban, swap) and installs k3s. Then the same Helm charts can be applied to it.

```bash
cd infra/ansible
ansible-galaxy collection install -r requirements.yml
ansible-playbook site.yml -e ansible_user=root
ansible-playbook site.yml
```
