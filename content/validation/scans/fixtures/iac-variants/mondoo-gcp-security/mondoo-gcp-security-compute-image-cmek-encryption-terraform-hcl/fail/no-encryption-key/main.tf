# Non-compliant: image has no image_encryption_key block (Google-managed keys).
resource "google_compute_image" "example" {
  name = "default-image"

  source_disk = "projects/my-project/zones/us-central1-a/disks/example-disk"
}
