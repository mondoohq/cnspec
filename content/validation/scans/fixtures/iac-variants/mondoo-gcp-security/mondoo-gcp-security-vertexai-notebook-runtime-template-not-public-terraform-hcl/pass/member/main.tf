# Compliant: access granted to a named group.
resource "google_colab_runtime_template_iam_member" "pass_example" {
  runtime_template = "projects/my-project/locations/us-central1/notebookRuntimeTemplates/123"
  location         = "us-central1"
  role   = "roles/viewer"
  member = "group:team@example.com"
}
