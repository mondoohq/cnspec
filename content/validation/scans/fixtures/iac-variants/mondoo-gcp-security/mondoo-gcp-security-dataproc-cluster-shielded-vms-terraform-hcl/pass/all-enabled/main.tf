# Compliant: shielded_instance_config enables secure boot, vTPM and integrity monitoring.
resource "google_dataproc_cluster" "compliant" {
  name   = "shielded-cluster"
  region = "us-central1"

  cluster_config {
    gce_cluster_config {
      zone = "us-central1-a"

      shielded_instance_config {
        enable_secure_boot          = true
        enable_vtpm                 = true
        enable_integrity_monitoring = true
      }
    }
  }
}
