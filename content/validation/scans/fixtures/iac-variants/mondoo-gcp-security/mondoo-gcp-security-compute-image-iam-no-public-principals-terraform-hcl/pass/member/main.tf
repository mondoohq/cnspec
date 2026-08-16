# Compliant: image IAM member grants access to a specific user, not the public.
resource "google_compute_image_iam_member" "example" {
  project = "my-project"
  image   = "example-image"
  role    = "roles/compute.imageUser"
  member  = "user:jane@example.com"
}
