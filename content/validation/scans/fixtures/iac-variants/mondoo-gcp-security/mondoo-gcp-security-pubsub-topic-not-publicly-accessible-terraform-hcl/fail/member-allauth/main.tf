# Non-compliant: IAM member grants access to allAuthenticatedUsers (public).
resource "google_pubsub_topic_iam_member" "fail_example" {
  project = "my-project"
  topic   = "my-topic"
  role    = "roles/pubsub.subscriber"
  member  = "allAuthenticatedUsers"
}
