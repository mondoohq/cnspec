# Non-compliant: integrity monitoring disabled in shielded_instance_config.
resource "google_dataproc_cluster" "no_integrity" {
  name   = "weak-cluster-2"
  region = "us-central1"

  cluster_config {
    gce_cluster_config {
      zone = "us-central1-a"

      shielded_instance_config {
        enable_secure_boot          = true
        enable_vtpm                 = true
        enable_integrity_monitoring = false
      }
    }
  }
}
