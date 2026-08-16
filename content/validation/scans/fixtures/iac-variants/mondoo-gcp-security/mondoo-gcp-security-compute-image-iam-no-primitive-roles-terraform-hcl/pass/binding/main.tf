# Compliant: image IAM binding grants a predefined, non-primitive role.
resource "google_compute_image_iam_binding" "example" {
  project = "my-project"
  image   = "example-image"
  role    = "roles/compute.imageUser"
  members = [
    "group:platform-team@example.com",
    "serviceAccount:builder@my-project.iam.gserviceaccount.com",
  ]
}
