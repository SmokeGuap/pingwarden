variable "kubeconfig_path" {
  type    = string
  default = "~/.kube/config"
}

variable "kube_context" {
  type        = string
  default     = "k3d-pingwarden"
  description = "cluster context"
}

variable "namespace" {
  type    = string
  default = "pingwarden"
}

variable "secret_name" {
  type    = string
  default = "pingwarden-secrets"
}

variable "image_repository" {
  type    = string
  default = "pingwarden"
}

variable "image_tag" {
  type    = string
  default = "dev"
}

variable "image_pull_policy" {
  type    = string
  default = "Never"
}

variable "alert_webhook_url" {
  type        = string
  default     = ""
  sensitive   = true
  description = "At least one channel (webhook or Telegram) is required; otherwise, the notifier will not start."
}

variable "telegram_bot_token" {
  type      = string
  default   = ""
  sensitive = true
}

variable "telegram_chat_id" {
  type    = string
  default = ""
}