# Non-compliant: image IAM binding grants the primitive roles/editor.
resource "google_compute_image_iam_binding" "example" {
  project = "my-project"
  image   = "example-image"
  role    = "roles/editor"
  members = [
    "group:platform-team@example.com",
  ]
}
