# Compliant: Cloud Run service template encrypts with a CMEK.
resource "google_cloud_run_v2_service" "svc" {
  name     = "cmek-svc"
  location = "us-central1"

  template {
    encryption_key = "projects/my-project/locations/us-central1/keyRings/run/cryptoKeys/cmek"

    containers {
      image = "us-docker.pkg.dev/cloudrun/container/hello"
    }
  }
}
