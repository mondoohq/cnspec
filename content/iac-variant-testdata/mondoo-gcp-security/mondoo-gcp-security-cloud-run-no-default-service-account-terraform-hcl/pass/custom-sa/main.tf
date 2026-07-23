# Compliant: the service runs as a dedicated, user-managed service account.
resource "google_cloud_run_v2_service" "pass_example" {
  name     = "web-app"
  location = "us-central1"

  template {
    service_account = "web-app-runner@my-project.iam.gserviceaccount.com"

    containers {
      image = "us-docker.pkg.dev/my-project/repo/web:latest"
    }
  }
}
