resource "google_cloudbuild_worker_pool" "fail" {
  name     = "public-pool"
  location = "us-central1"

  worker_config {
    disk_size_gb = 100
    machine_type = "e2-standard-4"
  }
}
