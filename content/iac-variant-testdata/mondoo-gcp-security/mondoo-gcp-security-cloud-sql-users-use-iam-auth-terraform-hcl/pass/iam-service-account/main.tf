# Compliant: a service account authenticates through Cloud IAM.
resource "google_sql_user" "iam_sa" {
  name     = "app-sa@my-project.iam"
  instance = google_sql_database_instance.main.name
  type     = "CLOUD_IAM_SERVICE_ACCOUNT"
}
