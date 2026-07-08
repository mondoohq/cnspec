# Compliant: detached attribute omitted (defaults to false).
resource "google_pubsub_subscription" "pass_example" {
  name  = "pass-subscription"
  topic = "projects/my-project/topics/my-topic"

  ack_deadline_seconds = 20
}
