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

resource "kubernetes_namespace_v1" "monitoring" {
  metadata {
    name = var.monitoring_namespace
  }
}

resource "random_password" "grafana" {
  length  = 24
  special = false
}

resource "kubernetes_secret_v1" "grafana_admin" {
  metadata {
    name      = "grafana-admin"
    namespace = kubernetes_namespace_v1.monitoring.metadata[0].name
  }
  data = {
    "admin-password" = random_password.grafana.result
  }
}

resource "helm_release" "monitoring" {
  name      = "monitoring"
  namespace = kubernetes_namespace_v1.monitoring.metadata[0].name
  chart     = "${path.module}/../../deploy/helm/monitoring"

  timeout = 600
  wait    = true

  set {
    name  = "appNamespace"
    value = var.namespace
  }
  set_sensitive {
    name  = "alertmanager.webhookURL"
    value = var.alert_webhook_url
  }

  set {
    name  = "chartHash"
    value = sha1(join("", [
      for f in sort(fileset("${path.module}/../../deploy/helm/monitoring", "**")) :
      filesha1("${path.module}/../../deploy/helm/monitoring/${f}")
    ]))
  }

  depends_on = [kubernetes_secret_v1.grafana_admin]
}

resource "helm_release" "loki" {
  name             = "loki"
  namespace        = var.logging_namespace
  create_namespace = true

  repository = "https://grafana.github.io/helm-charts"
  chart      = "loki"
  version    = var.loki_chart_version

  timeout = 600
  wait    = true

  values = [file("${path.module}/values/loki.yaml")]
}

resource "helm_release" "vector" {
  name      = "vector"
  namespace = var.logging_namespace

  repository = "https://helm.vector.dev"
  chart      = "vector"
  version    = var.vector_chart_version

  timeout = 600
  wait    = true

  values = [file("${path.module}/values/vector.yaml")]

  depends_on = [helm_release.loki]
}