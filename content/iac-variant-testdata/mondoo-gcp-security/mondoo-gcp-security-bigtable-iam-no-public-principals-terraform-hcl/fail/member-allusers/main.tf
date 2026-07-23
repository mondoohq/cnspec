# Non-compliant: IAM member grants access to allUsers (public).
resource "google_bigtable_instance_iam_member" "fail_example" {
  instance = "my-instance"
  role     = "roles/bigtable.reader"
  member   = "allUsers"
}
