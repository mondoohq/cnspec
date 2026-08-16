# Cloud Storage fixture for the tf-pass test bundle.
#
# Note: logging.tf already declares a `google_storage_bucket "log_bucket"` that
# satisfies the same checks. The bucket below adds extra coverage and exercises
# the IAM-binding / IAM-member checks for the
# mondoo-gcp-security-cloud-storage-bucket-not-anonymously-publicly-accessible
# terraform-hcl variant.

resource "google_storage_bucket" "data" {
  name          = "data-${random_id.rnd.hex}"
  location      = var.region
  force_destroy = false

  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  retention_policy {
    is_locked        = true
    retention_period = 2592000 # 30 days
  }

  soft_delete_policy {
    retention_duration_seconds = 1209600 # 14 days
  }

  encryption {
    default_kms_key_name = google_kms_crypto_key.key.id
  }
}

# Bind the bucket to a specific principal (NOT allUsers / allAuthenticatedUsers)
# so the not-anonymously-publicly-accessible-terraform-hcl filter triggers and
# the check exercises the IAM-member path.
resource "google_storage_bucket_iam_member" "data_admin" {
  bucket = google_storage_bucket.data.name
  role   = "roles/storage.objectViewer"
  member = "serviceAccount:${google_service_account.default.email}"
}

resource "google_storage_bucket_iam_binding" "data_viewer" {
  bucket = google_storage_bucket.data.name
  role   = "roles/storage.legacyBucketReader"
  members = [
    "serviceAccount:${google_service_account.default.email}",
  ]
}
