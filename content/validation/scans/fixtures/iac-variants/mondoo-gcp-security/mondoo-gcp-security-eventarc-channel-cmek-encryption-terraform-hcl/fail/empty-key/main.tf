# Non-compliant: crypto_key_name is an empty string.
resource "google_eventarc_channel" "primary" {
  location = "us-central1"
  name     = "my-channel"
  provider = google-beta

  crypto_key_name = ""
}
