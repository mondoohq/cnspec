# Non-compliant: image IAM member grants access to allUsers (public).
resource "google_compute_image_iam_member" "example" {
  project = "my-project"
  image   = "example-image"
  role    = "roles/compute.imageUser"
  member  = "allUsers"
}
