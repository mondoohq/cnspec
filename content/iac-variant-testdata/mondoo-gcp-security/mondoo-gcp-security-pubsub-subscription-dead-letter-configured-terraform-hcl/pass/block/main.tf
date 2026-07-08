# Compliant: subscription has a dead_letter_policy block.
resource "google_pubsub_subscription" "pass_example" {
  name  = "pass-subscription"
  topic = "projects/my-project/topics/my-topic"

  ack_deadline_seconds = 20

  dead_letter_policy {
    dead_letter_topic     = "projects/my-project/topics/my-dead-letter-topic"
    max_delivery_attempts = 10
  }
}
