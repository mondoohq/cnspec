# Compliant: ingress restricted to internal traffic and load balancer.
resource "google_cloud_run_v2_service" "svc" {
  name     = "lb-svc"
  location = "us-central1"
  ingress  = "INGRESS_TRAFFIC_INTERNAL_LOAD_BALANCER"

  template {
    containers {
      image = "us-docker.pkg.dev/cloudrun/container/hello"
    }
  }
}
