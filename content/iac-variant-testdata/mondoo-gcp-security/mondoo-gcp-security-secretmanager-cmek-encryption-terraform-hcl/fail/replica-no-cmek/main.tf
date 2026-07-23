# Non-compliant: user-managed replica has no customer_managed_encryption key.
resource "google_secret_manager_secret" "fail_example" {
  secret_id = "my-secret"

  replication {
    user_managed {
      replicas {
        location = "us-central1"
      }
    }
  }
}
