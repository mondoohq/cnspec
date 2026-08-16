# Non-compliant: no crypto_key_name, so the channel uses Google-managed keys.
resource "google_eventarc_channel" "primary" {
  location = "us-central1"
  name     = "my-channel"
  provider = google-beta
}
