# Compliant: the built-in "postgres" admin user is exempt.
resource "google_sql_user" "admin" {
  name     = "postgres"
  instance = google_sql_database_instance.main.name
  password = var.db_password
}
