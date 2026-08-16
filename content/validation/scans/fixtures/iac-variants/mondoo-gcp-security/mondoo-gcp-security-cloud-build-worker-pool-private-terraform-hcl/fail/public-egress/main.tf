resource "google_cloudbuild_worker_pool" "fail" {
  name     = "public-egress-pool"
  location = "us-central1"

  worker_config {
    disk_size_gb   = 100
    machine_type   = "e2-standard-4"
    no_external_ip = false
  }

  network_config {
    peered_network = "projects/my-project/global/networks/my-vpc"
    egress_option  = "PUBLIC_EGRESS"
  }
}
