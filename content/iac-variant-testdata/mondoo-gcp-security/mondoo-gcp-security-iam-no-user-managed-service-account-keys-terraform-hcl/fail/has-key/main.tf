# Non-compliant: a user-managed service account key is created.
resource "google_service_account" "app" {
  account_id   = "my-app-sa"
  display_name = "Application Service Account"
}

resource "google_service_account_key" "app_key" {
  service_account_id = google_service_account.app.name
}
