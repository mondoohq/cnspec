# Compliant: instance IAM member uses a predefined non-primitive role.
resource "google_spanner_instance_iam_member" "pass_example" {
  instance = "my-instance"
  role     = "roles/spanner.databaseAdmin"
  member   = "serviceAccount:app@my-project.iam.gserviceaccount.com"
}
