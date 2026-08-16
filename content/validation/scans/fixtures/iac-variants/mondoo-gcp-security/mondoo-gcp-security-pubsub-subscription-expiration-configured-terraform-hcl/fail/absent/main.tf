# Non-compliant: subscription has no expiration_policy block.
resource "google_pubsub_subscription" "fail_example" {
  name  = "fail-subscription"
  topic = "projects/my-project/topics/my-topic"

  ack_deadline_seconds = 20
}
