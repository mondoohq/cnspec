# Non-compliant: binding grants access to allAuthenticatedUsers (public).
resource "google_pubsub_topic_iam_binding" "fail_example" {
  project = "my-project"
  topic   = "my-topic"
  role    = "roles/pubsub.subscriber"

  members = [
    "allAuthenticatedUsers",
  ]
}
