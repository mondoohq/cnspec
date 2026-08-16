# Compliant: image encrypted with a customer-managed KMS key.
resource "google_compute_image" "example" {
  name = "cmek-image"

  source_disk = "projects/my-project/zones/us-central1-a/disks/example-disk"

  image_encryption_key {
    kms_key_self_link = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/image-key"
  }
}
