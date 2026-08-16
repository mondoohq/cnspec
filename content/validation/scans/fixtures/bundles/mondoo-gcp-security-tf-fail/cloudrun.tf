# Cloud Run fail fixture - every Cloud Run check should fail.
#
# - Ingress is INGRESS_TRAFFIC_ALL.
# - No CMEK encryption_key.
# - Uses the default Compute service account.
# - No vpc_access block.

resource "google_cloud_run_v2_service" "api" {
  name     = "fail-api-${random_id.suffix.hex}"
  location = "us-central1"
  ingress  = "INGRESS_TRAFFIC_ALL"

  template {
    # encryption_key intentionally absent
    service_account = "1234567890-compute@developer.gserviceaccount.com"
    # vpc_access intentionally absent

    containers {
      image = "us-docker.pkg.dev/cloudrun/container/hello"
    }
  }
}
