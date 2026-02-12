resource "kubernetes_namespace_v1" "app" {
  metadata {
    name = var.namespace
  }
}

resource "random_password" "postgres" {
  length  = 24
  special = false
}

resource "random_password" "api_token" {
  length  = 48
  special = false
}

resource "kubernetes_secret_v1" "app" {
  metadata {
    name      = var.secret_name
    namespace = kubernetes_namespace_v1.app.metadata[0].name
  }

  data = {
    POSTGRES_PASSWORD  = random_password.postgres.result
    API_TOKEN          = random_password.api_token.result
    ALERT_WEBHOOK_URL  = var.alert_webhook_url
    TELEGRAM_BOT_TOKEN = var.telegram_bot_token
    TELEGRAM_CHAT_ID   = var.telegram_chat_id
  }
}

resource "helm_release" "app" {
  name      = "pingwarden"
  namespace = kubernetes_namespace_v1.app.metadata[0].name
  chart     = "${path.module}/../../deploy/helm/pingwarden"

  timeout = 600
  wait    = true

  set {
    name  = "image.repository"
    value = var.image_repository
  }
  set {
    name  = "image.tag"
    value = var.image_tag
  }
  set {
    name  = "image.pullPolicy"
    value = var.image_pull_policy
  }
  set {
    name  = "secretName"
    value = var.secret_name
  }

  depends_on = [kubernetes_secret_v1.app]
}