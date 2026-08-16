# Compliant: the service template has a vpc_access block.
resource "google_cloud_run_v2_service" "pass_example" {
  name     = "web-app"
  location = "us-central1"

  template {
    containers {
      image = "us-docker.pkg.dev/my-project/repo/web:latest"
    }

    vpc_access {
      connector = "projects/my-project/locations/us-central1/connectors/my-connector"
      egress    = "ALL_TRAFFIC"
    }
  }
}
