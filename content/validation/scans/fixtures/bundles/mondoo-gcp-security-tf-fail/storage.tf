# Cloud Storage fail fixture - every storage check should fail.
#
# Note: compute.tf already declares a `google_storage_bucket "no-public-access"`
# that fails the public-access-prevention check. The resources below add more
# misconfigurations: no uniform bucket-level access, no CMEK, no retention
# policy, plus an IAM binding/member that grants allUsers.

resource "google_storage_bucket" "data" {
  name          = "fail-data-${random_id.suffix.hex}"
  location      = "US"
  force_destroy = true

  # uniform_bucket_level_access intentionally unset (defaults to false)
  # public_access_prevention intentionally unset
  # encryption block intentionally absent
  # retention_policy intentionally absent
}

# Anonymously / publicly accessible IAM bindings - fails
# the not-anonymously-publicly-accessible-terraform-hcl check.
resource "google_storage_bucket_iam_member" "public_member" {
  bucket = google_storage_bucket.data.name
  role   = "roles/storage.objectViewer"
  member = "allUsers"
}

resource "google_storage_bucket_iam_binding" "public_binding" {
  bucket = google_storage_bucket.data.name
  role   = "roles/storage.legacyBucketReader"
  members = [
    "allAuthenticatedUsers",
  ]
}
