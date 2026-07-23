# Compliant: push subscription delivers to an HTTPS endpoint.
resource "google_pubsub_subscription" "pass_example" {
  name  = "pass-subscription"
  topic = "projects/my-project/topics/my-topic"

  push_config {
    push_endpoint = "https://example.com/push"

    attributes = {
      x-goog-version = "v1"
    }
  }
}
