# Compliant: a service account is defined but no user-managed key is created.
resource "google_service_account" "app" {
  account_id   = "my-app-sa"
  display_name = "Application Service Account"
}
