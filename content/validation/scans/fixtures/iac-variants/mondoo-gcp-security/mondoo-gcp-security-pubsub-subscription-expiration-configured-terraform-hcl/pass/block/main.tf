# Compliant: subscription has an expiration_policy block.
resource "google_pubsub_subscription" "pass_example" {
  name  = "pass-subscription"
  topic = "projects/my-project/topics/my-topic"

  expiration_policy {
    ttl = "2678400s"
  }
}
