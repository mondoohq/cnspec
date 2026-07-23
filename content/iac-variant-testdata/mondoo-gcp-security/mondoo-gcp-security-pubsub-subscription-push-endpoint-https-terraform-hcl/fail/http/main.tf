# Non-compliant: push endpoint uses plaintext HTTP.
resource "google_pubsub_subscription" "fail_example" {
  name  = "fail-subscription"
  topic = "projects/my-project/topics/my-topic"

  push_config {
    push_endpoint = "http://example.com/push"
  }
}
