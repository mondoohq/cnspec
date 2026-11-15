# Non-compliant: an empty ttl makes the subscription never expire.
resource "google_pubsub_subscription" "fail_example" {
  name  = "fail-subscription"
  topic = "projects/my-project/topics/my-topic"

  expiration_policy {
    ttl = ""
  }
}
