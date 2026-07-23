# Non-compliant: container image has no tag (implicitly :latest).
resource "google_cloud_run_v2_job" "job" {
  name     = "untagged-job"
  location = "us-central1"

  template {
    template {
      containers {
        image = "us-docker.pkg.dev/my-project/repo/worker"
      }
    }
  }
}
