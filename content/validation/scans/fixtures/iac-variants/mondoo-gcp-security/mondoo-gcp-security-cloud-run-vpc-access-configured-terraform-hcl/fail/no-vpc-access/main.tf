# Non-compliant: the service template has no vpc_access block.
resource "google_cloud_run_v2_service" "fail_example" {
  name     = "web-app"
  location = "us-central1"

  template {
    containers {
      image = "us-docker.pkg.dev/my-project/repo/web:latest"
    }
  }
}
