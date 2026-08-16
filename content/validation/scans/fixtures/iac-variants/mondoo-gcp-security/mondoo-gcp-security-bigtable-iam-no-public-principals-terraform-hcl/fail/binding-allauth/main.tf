# Non-compliant: IAM binding includes allAuthenticatedUsers (public).
resource "google_bigtable_instance_iam_binding" "fail_example" {
  instance = "my-instance"
  role     = "roles/bigtable.reader"

  members = [
    "group:analysts@example.com",
    "allAuthenticatedUsers",
  ]
}
