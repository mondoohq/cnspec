# Compliant: the Eventarc channel is encrypted with a customer-managed key.
resource "google_eventarc_channel" "primary" {
  location = "us-central1"
  name     = "my-channel"
  provider = google-beta

  crypto_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
}
