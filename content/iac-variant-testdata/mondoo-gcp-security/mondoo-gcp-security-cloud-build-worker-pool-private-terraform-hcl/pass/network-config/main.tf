resource "google_cloudbuild_worker_pool" "pass" {
  name     = "private-pool"
  location = "us-central1"

  worker_config {
    disk_size_gb   = 100
    machine_type   = "e2-standard-4"
    no_external_ip = true
  }

  network_config {
    peered_network          = "projects/my-project/global/networks/my-vpc"
    peered_network_ip_range = "/29"
    egress_option           = "NO_PUBLIC_EGRESS"
  }
}
