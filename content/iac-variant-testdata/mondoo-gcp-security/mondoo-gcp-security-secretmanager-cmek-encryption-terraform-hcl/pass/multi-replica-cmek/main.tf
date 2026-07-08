# Compliant: every user-managed replica is protected with a CMEK.
resource "google_secret_manager_secret" "pass_example" {
  secret_id = "my-secret"

  replication {
    user_managed {
      replicas {
        location = "us-central1"
        customer_managed_encryption {
          kms_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/key-a"
        }
      }
      replicas {
        location = "us-east1"
        customer_managed_encryption {
          kms_key_name = "projects/my-project/locations/us-east1/keyRings/my-ring/cryptoKeys/key-b"
        }
      }
    }
  }
}
