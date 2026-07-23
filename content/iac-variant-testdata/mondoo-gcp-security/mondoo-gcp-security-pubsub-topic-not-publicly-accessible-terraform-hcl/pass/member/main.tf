# Compliant: IAM member is a specific user, not a public principal.
resource "google_pubsub_topic_iam_member" "pass_example" {
  project = "my-project"
  topic   = "my-topic"
  role    = "roles/pubsub.subscriber"
  member  = "user:alice@example.com"
}
