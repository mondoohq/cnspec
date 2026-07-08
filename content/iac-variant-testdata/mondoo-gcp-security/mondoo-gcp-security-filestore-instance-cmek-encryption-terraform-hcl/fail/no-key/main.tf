# Non-compliant: no kms_key_name, so the instance uses Google-managed keys.
resource "google_filestore_instance" "primary" {
  name     = "my-filestore"
  location = "us-central1-b"
  tier     = "ENTERPRISE"

  file_shares {
    capacity_gb = 1024
    name        = "share1"
  }

  networks {
    network = "default"
    modes   = ["MODE_IPV4"]
  }
}
