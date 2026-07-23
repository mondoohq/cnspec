# Compliant: every cluster block references a CMEK key.
resource "google_bigtable_instance" "pass_example" {
  name = "pass-instance"

  cluster {
    cluster_id   = "pass-cluster"
    zone         = "us-central1-b"
    num_nodes    = 3
    storage_type = "SSD"
    kms_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
  }
}
