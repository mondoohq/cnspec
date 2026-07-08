# Non-compliant: secure boot disabled in shielded_instance_config.
resource "google_dataproc_cluster" "no_secure_boot" {
  name   = "weak-cluster"
  region = "us-central1"

  cluster_config {
    gce_cluster_config {
      zone = "us-central1-a"

      shielded_instance_config {
        enable_secure_boot          = false
        enable_vtpm                 = true
        enable_integrity_monitoring = true
      }
    }
  }
}
