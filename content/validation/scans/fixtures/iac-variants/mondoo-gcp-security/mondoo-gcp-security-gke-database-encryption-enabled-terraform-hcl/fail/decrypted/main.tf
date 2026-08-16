resource "google_container_cluster" "primary" {
  name     = "primary"
  location = "us-central1"

  database_encryption {
    state = "DECRYPTED"
  }
}
