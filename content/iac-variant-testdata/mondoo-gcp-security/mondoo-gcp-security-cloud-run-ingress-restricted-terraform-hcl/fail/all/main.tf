# Non-compliant: ingress allows all traffic from the internet.
resource "google_cloud_run_v2_service" "svc" {
  name     = "public-svc"
  location = "us-central1"
  ingress  = "INGRESS_TRAFFIC_ALL"

  template {
    containers {
      image = "us-docker.pkg.dev/cloudrun/container/hello"
    }
  }
}
