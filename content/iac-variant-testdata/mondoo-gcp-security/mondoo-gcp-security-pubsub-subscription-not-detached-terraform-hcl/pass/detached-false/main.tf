# Compliant: subscription is explicitly not detached.
resource "google_pubsub_subscription" "pass_example" {
  name     = "pass-subscription"
  topic    = "projects/my-project/topics/my-topic"
  detached = false
}
