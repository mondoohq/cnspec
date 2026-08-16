# Naming a customer-managed key selects who holds the key for transparent data
# encryption. It does not turn the feature on, so this service is still unencrypted.
resource "clickhouse_service" "analytics" {
  name           = "analytics"
  cloud_provider = "aws"
  region         = "us-east-1"
  password_wo    = var.clickhouse_default_password

  ip_access = [
    {
      source      = "203.0.113.0/24"
      description = "corporate egress"
    }
  ]

  encryption_key                     = var.kms_key_arn
  encryption_assumed_role_identifier = var.kms_role_arn
}
