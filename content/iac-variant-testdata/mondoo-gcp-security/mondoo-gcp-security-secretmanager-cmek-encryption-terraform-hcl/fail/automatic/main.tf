# Non-compliant: automatic replication uses Google-managed keys, not CMEK.
resource "google_secret_manager_secret" "fail_example" {
  secret_id = "my-secret"

  replication {
    auto {}
  }
}
