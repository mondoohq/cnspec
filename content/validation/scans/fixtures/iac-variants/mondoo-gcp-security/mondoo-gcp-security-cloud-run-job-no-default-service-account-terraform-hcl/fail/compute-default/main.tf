# Non-compliant: explicitly set to the default Compute Engine service account.
resource "google_cloud_run_v2_job" "job" {
  name     = "compute-sa-job"
  location = "us-central1"

  template {
    template {
      service_account = "123456789012-compute@developer.gserviceaccount.com"

      containers {
        image = "us-docker.pkg.dev/cloudrun/container/job"
      }
    }
  }
}
