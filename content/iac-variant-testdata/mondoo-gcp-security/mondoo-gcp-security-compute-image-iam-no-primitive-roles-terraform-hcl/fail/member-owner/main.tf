# Non-compliant: image IAM member grants the primitive roles/owner.
resource "google_compute_image_iam_member" "example" {
  project = "my-project"
  image   = "example-image"
  role    = "roles/owner"
  member  = "user:jane@example.com"
}
