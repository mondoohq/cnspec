# Compliant: ingress restricted to internal traffic only.
resource "google_cloud_run_v2_service" "svc" {
  name     = "internal-svc"
  location = "us-central1"
  ingress  = "INGRESS_TRAFFIC_INTERNAL_ONLY"

  template {
    containers {
      image = "us-docker.pkg.dev/cloudrun/container/hello"
    }
  }
}
