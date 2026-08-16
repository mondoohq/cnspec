# Compliant: binding members are specific principals, not public.
resource "google_pubsub_topic_iam_binding" "pass_example" {
  project = "my-project"
  topic   = "my-topic"
  role    = "roles/pubsub.publisher"

  members = [
    "group:publishers@example.com",
    "serviceAccount:app@my-project.iam.gserviceaccount.com",
  ]
}
