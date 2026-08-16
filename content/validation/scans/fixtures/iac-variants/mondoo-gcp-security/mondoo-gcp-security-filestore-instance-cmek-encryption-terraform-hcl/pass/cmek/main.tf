# Compliant: the Filestore instance is encrypted with a customer-managed key.
resource "google_filestore_instance" "primary" {
  name     = "my-filestore"
  location = "us-central1-b"
  tier     = "ENTERPRISE"

  kms_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"

  file_shares {
    capacity_gb = 1024
    name        = "share1"
  }

  networks {
    network = "default"
    modes   = ["MODE_IPV4"]
  }
}
