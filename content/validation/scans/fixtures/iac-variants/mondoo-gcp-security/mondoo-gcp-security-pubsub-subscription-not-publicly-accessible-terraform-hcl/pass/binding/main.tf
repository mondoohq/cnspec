# Compliant: IAM binding grants access only to a specific service account.
resource "google_pubsub_subscription_iam_binding" "pass_example" {
  subscription = "projects/my-project/subscriptions/my-subscription"
  role         = "roles/pubsub.subscriber"
  members = [
    "serviceAccount:consumer@my-project.iam.gserviceaccount.com",
    "user:alice@example.com",
  ]
}
