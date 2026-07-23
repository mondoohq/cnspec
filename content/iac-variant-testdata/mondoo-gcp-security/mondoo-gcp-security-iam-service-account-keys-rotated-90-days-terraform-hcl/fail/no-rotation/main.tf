# Non-compliant: the service account key has no rotation schedule at all.
resource "google_service_account" "app" {
  account_id = "my-app-sa"
}

resource "google_service_account_key" "app_key" {
  service_account_id = google_service_account.app.name
}
