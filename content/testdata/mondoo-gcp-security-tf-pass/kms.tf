resource "google_kms_key_ring" "keyring" {
  name     = "keyring-example-${random_id.rnd.hex}"
  location = "global"
}

resource "google_kms_crypto_key" "key" {
  name            = "crypto-key-example-${random_id.rnd.hex}"
  key_ring        = google_kms_key_ring.keyring.id
  rotation_period = "7776000s"
  lifecycle {
    prevent_destroy = false
  }
}

# Get the current project's default service account for IAM binding
data "google_project" "current" {}

data "google_iam_policy" "admin" {
  binding {
    role = "roles/cloudkms.cryptoKeyEncrypter"

    members = [
      "serviceAccount:${data.google_project.current.number}-compute@developer.gserviceaccount.com",
    ]
  }
}

resource "google_kms_crypto_key_iam_policy" "crypto_key" {
  crypto_key_id = google_kms_crypto_key.key.id
  policy_data   = data.google_iam_policy.admin.policy_data
}