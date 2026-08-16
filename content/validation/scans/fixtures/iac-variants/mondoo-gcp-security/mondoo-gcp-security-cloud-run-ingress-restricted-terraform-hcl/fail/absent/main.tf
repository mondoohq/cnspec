# Non-compliant: ingress omitted (defaults to allow all).
resource "google_cloud_run_v2_service" "svc" {
  name     = "default-svc"
  location = "us-central1"

  template {
    containers {
      image = "us-docker.pkg.dev/cloudrun/container/hello"
    }
  }
}
