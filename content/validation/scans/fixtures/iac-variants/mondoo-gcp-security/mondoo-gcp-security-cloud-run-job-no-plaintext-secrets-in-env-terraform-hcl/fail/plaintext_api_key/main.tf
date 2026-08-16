resource "google_cloud_run_v2_job" "default" {
  name     = "worker-job"
  location = "us-central1"

  template {
    template {
      containers {
        image = "us-docker.pkg.dev/cloudrun/container/hello"

        env {
          name  = "SERVICE_URL"
          value = "https://example.com"
        }

        env {
          name  = "API_KEY"
          value = "AIzaSyExamplePlaintextKeyValue1234567890"
        }
      }
    }
  }
}
