# Compliant: IAM binding members are specific principals, not public.
resource "google_bigtable_instance_iam_binding" "pass_example" {
  instance = "my-instance"
  role     = "roles/bigtable.reader"

  members = [
    "group:analysts@example.com",
    "serviceAccount:app@my-project.iam.gserviceaccount.com",
  ]
}
