# Compliant: cluster has an encryption_config block with a KMS key.
resource "google_alloydb_cluster" "pass_example" {
  cluster_id = "pass-cluster"
  location   = "us-central1"

  encryption_config {
    kms_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
  }
}
