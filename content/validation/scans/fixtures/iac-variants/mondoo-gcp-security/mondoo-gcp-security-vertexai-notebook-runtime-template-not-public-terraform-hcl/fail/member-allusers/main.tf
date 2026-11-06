# Non-compliant: member grants access to allUsers.
resource "google_colab_runtime_template_iam_member" "fail_example" {
  runtime_template = "projects/my-project/locations/us-central1/notebookRuntimeTemplates/123"
  location         = "us-central1"
  role   = "roles/viewer"
  member = "allUsers"
}
