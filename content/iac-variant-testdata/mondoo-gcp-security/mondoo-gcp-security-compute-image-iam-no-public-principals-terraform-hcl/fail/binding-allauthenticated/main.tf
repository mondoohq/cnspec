# Non-compliant: image IAM binding includes allAuthenticatedUsers (public).
resource "google_compute_image_iam_binding" "example" {
  project = "my-project"
  image   = "example-image"
  role    = "roles/compute.imageUser"
  members = [
    "group:platform-team@example.com",
    "allAuthenticatedUsers",
  ]
}
