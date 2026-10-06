resource "exoscale_sks_cluster" "prod" {
  zone = "ch-gva-2"
  name = "prod"

  audit {
    enabled      = false
    endpoint     = "https://audit.example.com/k8s"
    bearer_token = var.audit_bearer_token
  }
}

variable "audit_bearer_token" {
  type      = string
  sensitive = true
}
