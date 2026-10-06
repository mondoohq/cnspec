# An audit block without `enabled` creates a cluster without audit logging:
# the provider only sends the audit configuration when enabled is true.
resource "exoscale_sks_cluster" "prod" {
  zone          = "ch-gva-2"
  name          = "prod"
  service_level = "pro"

  audit {
    endpoint     = "https://audit.example.com/k8s"
    bearer_token = var.audit_bearer_token
  }
}

variable "audit_bearer_token" {
  type      = string
  sensitive = true
}
