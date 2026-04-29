# Cloud Functions fixture for the tf-pass test bundle.
#
# The function uses a custom service account, restricted ingress, a VPC
# connector, CMEK, no plaintext secrets in env vars, and a user-managed Artifact
# Registry repository.

resource "google_service_account" "function_sa" {
  account_id   = "function-sa-${random_id.rnd.hex}"
  display_name = "Cloud Function Service Account"
}

resource "google_artifact_registry_repository" "functions" {
  location      = var.region
  repository_id = "functions-${random_id.rnd.hex}"
  format        = "DOCKER"
  kms_key_name  = google_kms_crypto_key.key.id
}

resource "google_storage_bucket_object" "function_source" {
  name   = "function-source.zip"
  bucket = google_storage_bucket.data.name
  source = "/dev/null"
}

resource "google_cloudfunctions2_function" "api" {
  name     = "api-fn-${random_id.rnd.hex}"
  location = var.region

  # MQL inspects arguments.kms_key_name, .ingress_settings, .vpc_connector,
  # .service_account_email at the top level of the resource block.
  kms_key_name          = google_kms_crypto_key.key.id
  ingress_settings      = "ALLOW_INTERNAL_ONLY"
  vpc_connector         = google_vpc_access_connector.run_connector.id
  service_account_email = google_service_account.function_sa.email

  build_config {
    runtime           = "python312"
    entry_point       = "main"
    docker_repository = google_artifact_registry_repository.functions.id

    source {
      storage_source {
        bucket = google_storage_bucket.data.name
        object = google_storage_bucket_object.function_source.name
      }
    }
  }

  service_config {
    service_account_email = google_service_account.function_sa.email
    ingress_settings      = "ALLOW_INTERNAL_ONLY"
    vpc_connector         = google_vpc_access_connector.run_connector.id

    environment_variables = {
      LOG_LEVEL = "info"
      REGION    = var.region
    }
  }
}
