# Compliant: IAM member grants access to a single named principal.
resource "google_pubsub_subscription_iam_member" "pass_example" {
  subscription = "projects/my-project/subscriptions/my-subscription"
  role         = "roles/pubsub.subscriber"
  member       = "serviceAccount:consumer@my-project.iam.gserviceaccount.com"
}
