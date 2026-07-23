# Non-compliant: subscription is detached, stopping message delivery.
resource "google_pubsub_subscription" "fail_example" {
  name     = "fail-subscription"
  topic    = "projects/my-project/topics/my-topic"
  detached = true
}
