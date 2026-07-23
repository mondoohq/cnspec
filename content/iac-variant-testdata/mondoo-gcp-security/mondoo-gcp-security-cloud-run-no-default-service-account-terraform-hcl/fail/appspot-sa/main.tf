# Non-compliant: the service uses the default App Engine service account.
resource "google_cloud_run_v2_service" "fail_example" {
  name     = "web-app"
  location = "us-central1"

  template {
    service_account = "my-project@appspot.gserviceaccount.com"

    containers {
      image = "us-docker.pkg.dev/my-project/repo/web:latest"
    }
  }
}
