# Non-compliant: kms_key_name is an empty string.
resource "google_filestore_instance" "primary" {
  name     = "my-filestore"
  location = "us-central1-b"
  tier     = "ENTERPRISE"

  kms_key_name = ""

  file_shares {
    capacity_gb = 1024
    name        = "share1"
  }

  networks {
    network = "default"
    modes   = ["MODE_IPV4"]
  }
}
