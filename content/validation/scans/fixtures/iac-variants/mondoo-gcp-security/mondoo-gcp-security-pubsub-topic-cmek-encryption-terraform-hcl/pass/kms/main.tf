# Compliant: topic encrypted with a customer-managed KMS key.
resource "google_pubsub_topic" "pass_example" {
  name         = "pass-topic"
  kms_key_name = "projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key"
}
