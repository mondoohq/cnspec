# Non-compliant: the rotation schedule is 180 days, exceeding the 90-day maximum.
resource "google_service_account" "app" {
  account_id = "my-app-sa"
}

resource "time_rotating" "sa_key_rotation" {
  rotation_days = 180
}

resource "google_service_account_key" "app_key" {
  service_account_id = google_service_account.app.name

  keepers = {
    rotation_time = time_rotating.sa_key_rotation.rotation_rfc3339
  }
}
