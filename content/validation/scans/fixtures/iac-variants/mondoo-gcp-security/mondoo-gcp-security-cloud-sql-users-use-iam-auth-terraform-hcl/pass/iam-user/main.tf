# Compliant: user authenticates through Cloud IAM.
resource "google_sql_user" "iam" {
  name     = "jane@example.com"
  instance = google_sql_database_instance.main.name
  type     = "CLOUD_IAM_USER"
}
