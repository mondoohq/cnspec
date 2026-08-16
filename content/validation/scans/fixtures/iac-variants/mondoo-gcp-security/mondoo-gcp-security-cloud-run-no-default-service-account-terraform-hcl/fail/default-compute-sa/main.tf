# Non-compliant: the service explicitly uses the default compute service account.
resource "google_cloud_run_v2_service" "fail_example" {
  name     = "web-app"
  location = "us-central1"

  template {
    service_account = "123456789012-compute@developer.gserviceaccount.com"

    containers {
      image = "us-docker.pkg.dev/my-project/repo/web:latest"
    }
  }
}
