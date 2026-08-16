# Non-compliant: IAM binding grants subscriber role to allUsers.
resource "google_pubsub_subscription_iam_binding" "fail_example" {
  subscription = "projects/my-project/subscriptions/my-subscription"
  role         = "roles/pubsub.subscriber"
  members = [
    "serviceAccount:consumer@my-project.iam.gserviceaccount.com",
    "allUsers",
  ]
}
