# Compliant: user-managed replication with a customer-managed encryption key.
resource "google_secret_manager_secret" "pass_example" {
  secret_id = "my-secret"

  replication {
    user_managed {
      replicas {
        location = "us-central1"
        customer_managed_encryption {
          kms_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
        }
      }
    }
  }
}
