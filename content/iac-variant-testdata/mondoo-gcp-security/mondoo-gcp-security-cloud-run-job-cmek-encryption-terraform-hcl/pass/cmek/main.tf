# Compliant: Cloud Run job execution template encrypts with a CMEK.
resource "google_cloud_run_v2_job" "job" {
  name     = "cmek-job"
  location = "us-central1"

  template {
    template {
      encryption_key = "projects/my-project/locations/us-central1/keyRings/run/cryptoKeys/cmek"

      containers {
        image = "us-docker.pkg.dev/cloudrun/container/job"
      }
    }
  }
}
