# Non-compliant: topic has no schema_settings block, so messages are unvalidated.
resource "google_pubsub_topic" "fail_example" {
  name = "my-topic"

  message_retention_duration = "86600s"
}
