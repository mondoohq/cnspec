# Non-compliant: internal_ip_only explicitly disabled, so nodes get public IPs.
resource "google_dataproc_cluster" "public_ips" {
  name   = "public-cluster"
  region = "us-central1"

  cluster_config {
    gce_cluster_config {
      zone             = "us-central1-a"
      internal_ip_only = false
    }
  }
}
