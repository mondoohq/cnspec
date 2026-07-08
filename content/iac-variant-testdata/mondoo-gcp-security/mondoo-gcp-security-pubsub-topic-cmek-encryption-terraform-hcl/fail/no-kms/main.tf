# Non-compliant: topic has no kms_key_name (Google-managed encryption only).
resource "google_pubsub_topic" "fail_example" {
  name = "fail-topic"

  labels = {
    environment = "production"
  }
}
