resource "google_container_cluster" "primary" {
  name     = "primary"
  location = "us-central1"

  database_encryption {
    state    = "ENCRYPTED"
    key_name = "projects/my-project/locations/us-central1/keyRings/gke/cryptoKeys/etcd"
  }
}
