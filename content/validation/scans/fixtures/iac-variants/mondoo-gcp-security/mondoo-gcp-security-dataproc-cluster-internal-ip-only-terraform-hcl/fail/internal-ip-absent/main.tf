# Non-compliant: gce_cluster_config present but internal_ip_only omitted (defaults to public IPs).
resource "google_dataproc_cluster" "default_ips" {
  name   = "default-cluster"
  region = "us-central1"

  cluster_config {
    gce_cluster_config {
      zone       = "us-central1-a"
      subnetwork = "projects/my-project/regions/us-central1/subnetworks/default"
    }
  }
}
