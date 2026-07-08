# Non-compliant: uniform bucket-level access constraint not enforced.
resource "google_org_policy_policy" "ubla" {
  name   = "projects/my-project/policies/storage.uniformBucketLevelAccess"
  parent = "projects/my-project"

  spec {
    rules {
      enforce = "FALSE"
    }
  }
}
