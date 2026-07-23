# Non-compliant: image_encryption_key block present but no kms_key_self_link set.
resource "google_compute_image" "example" {
  name = "no-kms-image"

  source_disk = "projects/my-project/zones/us-central1-a/disks/example-disk"

  image_encryption_key {
    kms_key_service_account = "image-encryptor@my-project.iam.gserviceaccount.com"
  }
}
