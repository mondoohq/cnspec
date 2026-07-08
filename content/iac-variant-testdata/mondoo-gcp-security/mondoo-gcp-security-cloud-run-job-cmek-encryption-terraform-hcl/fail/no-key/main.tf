# Non-compliant: job execution template has no encryption_key.
resource "google_cloud_run_v2_job" "job" {
  name     = "gmek-job"
  location = "us-central1"

  template {
    template {
      containers {
        image = "us-docker.pkg.dev/cloudrun/container/job"
      }
    }
  }
}
