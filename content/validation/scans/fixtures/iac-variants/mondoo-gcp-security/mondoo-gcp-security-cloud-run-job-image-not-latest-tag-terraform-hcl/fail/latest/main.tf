# Non-compliant: container image uses the mutable :latest tag.
resource "google_cloud_run_v2_job" "job" {
  name     = "latest-job"
  location = "us-central1"

  template {
    template {
      containers {
        image = "us-docker.pkg.dev/my-project/repo/worker:latest"
      }
    }
  }
}
