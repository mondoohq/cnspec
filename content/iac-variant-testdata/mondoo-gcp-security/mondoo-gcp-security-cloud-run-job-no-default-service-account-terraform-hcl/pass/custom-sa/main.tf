# Compliant: job runs as a dedicated, least-privilege service account.
resource "google_cloud_run_v2_job" "job" {
  name     = "custom-sa-job"
  location = "us-central1"

  template {
    template {
      service_account = "worker-runner@my-project.iam.gserviceaccount.com"

      containers {
        image = "us-docker.pkg.dev/cloudrun/container/job"
      }
    }
  }
}
