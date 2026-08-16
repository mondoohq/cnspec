resource "clickhouse_service" "analytics" {
  name           = "analytics"
  cloud_provider = "aws"
  region         = "us-east-1"
  password_wo    = var.clickhouse_default_password

  ip_access = [
    {
      source      = "0.0.0.0/0"
      description = "anywhere"
    }
  ]
}
