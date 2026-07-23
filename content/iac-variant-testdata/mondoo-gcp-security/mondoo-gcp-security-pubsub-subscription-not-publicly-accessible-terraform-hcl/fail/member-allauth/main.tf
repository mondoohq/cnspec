# Non-compliant: IAM member grants subscriber role to allAuthenticatedUsers.
resource "google_pubsub_subscription_iam_member" "fail_example" {
  subscription = "projects/my-project/subscriptions/my-subscription"
  role         = "roles/pubsub.subscriber"
  member       = "allAuthenticatedUsers"
}
