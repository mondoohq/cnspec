# Compliant: topic pins message storage to allowed regions.
resource "google_pubsub_topic" "pass_example" {
  name = "pass-topic"

  message_storage_policy {
    allowed_persistence_regions = [
      "europe-west1",
      "europe-west3",
    ]
  }
}
