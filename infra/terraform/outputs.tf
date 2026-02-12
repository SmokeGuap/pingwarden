output "api_token" {
  value     = random_password.api_token.result
  sensitive = true
}

output "port_forward" {
  value = "kubectl -n ${var.namespace} port-forward svc/pingwarden-api 8080:8080"
}