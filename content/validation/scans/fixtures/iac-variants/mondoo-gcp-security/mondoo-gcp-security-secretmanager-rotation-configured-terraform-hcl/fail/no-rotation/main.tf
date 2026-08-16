# Non-compliant: secret has no rotation block configured.
resource "google_secret_manager_secret" "fail_example" {
  secret_id = "my-secret"

  replication {
    auto {}
  }
}
