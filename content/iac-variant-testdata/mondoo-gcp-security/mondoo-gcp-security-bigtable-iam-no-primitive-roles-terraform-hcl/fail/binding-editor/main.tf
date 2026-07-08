# Non-compliant: IAM binding grants the primitive roles/editor role.
resource "google_bigtable_instance_iam_binding" "fail_example" {
  instance = "my-instance"
  role     = "roles/editor"

  members = [
    "group:developers@example.com",
  ]
}
