# Compliant: binding lists only named principals.
resource "google_colab_runtime_template_iam_binding" "pass_example" {
  runtime_template = "projects/my-project/locations/us-central1/notebookRuntimeTemplates/123"
  location         = "us-central1"
  role = "roles/viewer"

  members = [
    "group:team@example.com",
    "serviceAccount:caller@my-project.iam.gserviceaccount.com",
  ]
}
