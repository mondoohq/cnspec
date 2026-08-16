# Non-compliant: an application user relies on built-in password auth.
resource "google_sql_user" "app" {
  name     = "appuser"
  instance = google_sql_database_instance.main.name
  type     = "BUILT_IN"
  password = var.db_password
}
