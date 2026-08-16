# Compliant: image IAM member grants a predefined, non-primitive role.
resource "google_compute_image_iam_member" "example" {
  project = "my-project"
  image   = "example-image"
  role    = "roles/compute.imageUser"
  member  = "user:jane@example.com"
}
