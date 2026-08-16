# Compliant: the instance is encrypted with a customer-managed KMS key.
resource "google_sql_database_instance" "pass_example" {
  name                = "app-db"
  database_version    = "POSTGRES_15"
  region              = "us-central1"
  encryption_key_name = "projects/my-project/locations/us-central1/keyRings/sql-ring/cryptoKeys/sql-key"

  settings {
    tier = "db-custom-2-7680"
  }
}
