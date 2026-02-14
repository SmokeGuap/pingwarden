output "api_token" {
  value     = random_password.api_token.result
  sensitive = true
}

output "port_forward" {
  value = "kubectl -n ${var.namespace} port-forward svc/pingwarden-api 8080:8080"
}

output "grafana_password" {
  value     = random_password.grafana.result
  sensitive = true
}

output "monitoring_port_forwards" {
  value = <<-EOT
    kubectl -n ${var.monitoring_namespace} port-forward svc/grafana 3000:3000
    kubectl -n ${var.monitoring_namespace} port-forward svc/prometheus 9091:9090
    kubectl -n ${var.monitoring_namespace} port-forward svc/alertmanager 9093:9093
  EOT
}