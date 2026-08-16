# Compliant: container image pinned to an explicit version tag.
resource "google_cloud_run_v2_job" "job" {
  name     = "pinned-job"
  location = "us-central1"

  template {
    template {
      containers {
        image = "us-docker.pkg.dev/my-project/repo/worker:v1.4.2"
      }
    }
  }
}
