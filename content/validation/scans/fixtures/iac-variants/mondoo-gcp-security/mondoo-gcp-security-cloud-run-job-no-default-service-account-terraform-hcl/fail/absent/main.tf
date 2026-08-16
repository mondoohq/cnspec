# Non-compliant: no service_account set (uses the default compute account).
resource "google_cloud_run_v2_job" "job" {
  name     = "default-sa-job"
  location = "us-central1"

  template {
    template {
      containers {
        image = "us-docker.pkg.dev/cloudrun/container/job"
      }
    }
  }
}
