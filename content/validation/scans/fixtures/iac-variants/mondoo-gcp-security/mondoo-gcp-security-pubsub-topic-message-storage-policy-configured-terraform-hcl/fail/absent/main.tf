# Non-compliant: topic has no message_storage_policy block.
resource "google_pubsub_topic" "fail_example" {
  name = "fail-topic"

  labels = {
    environment = "production"
  }
}
