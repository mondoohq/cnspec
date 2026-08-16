# Non-compliant: binding grants access to allUsers (public).
resource "google_pubsub_topic_iam_binding" "fail_example" {
  project = "my-project"
  topic   = "my-topic"
  role    = "roles/pubsub.publisher"

  members = [
    "group:publishers@example.com",
    "allUsers",
  ]
}
