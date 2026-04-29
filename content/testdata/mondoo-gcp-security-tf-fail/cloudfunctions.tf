# Cloud Functions fail fixture - every Cloud Functions check should fail.
#
# - ingress_settings is ALLOW_ALL.
# - service_account_email is the default Compute service account.
# - No vpc_connector.
# - No kms_key_name.
# - environment_variables include plaintext secrets.
# - No build_config.docker_repository (uses Container Registry default).

resource "google_cloudfunctions2_function" "api" {
  name     = "fail-api-fn-${random_id.suffix.hex}"
  location = "us-central1"

  ingress_settings      = "ALLOW_ALL"
  service_account_email = "1234567890-compute@developer.gserviceaccount.com"
  # vpc_connector intentionally absent
  # kms_key_name intentionally absent

  build_config {
    runtime     = "python312"
    entry_point = "main"
    # docker_repository intentionally absent
  }

  service_config {
    service_account_email = "1234567890-compute@developer.gserviceaccount.com"
    ingress_settings      = "ALLOW_ALL"

    environment_variables = {
      DB_PASSWORD = "hunter2"
      API_KEY     = "sk-fail"
    }
  }
}
