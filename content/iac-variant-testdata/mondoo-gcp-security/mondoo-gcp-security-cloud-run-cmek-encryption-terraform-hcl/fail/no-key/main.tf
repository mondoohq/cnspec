# Non-compliant: template has no encryption_key (Google-managed encryption).
resource "google_cloud_run_v2_service" "svc" {
  name     = "gmek-svc"
  location = "us-central1"

  template {
    containers {
      image = "us-docker.pkg.dev/cloudrun/container/hello"
    }
  }
}
