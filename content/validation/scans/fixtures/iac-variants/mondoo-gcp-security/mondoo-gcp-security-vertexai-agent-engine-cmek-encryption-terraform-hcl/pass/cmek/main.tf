# Compliant: agent encrypted with a customer-managed key.
resource "google_vertex_ai_reasoning_engine" "pass_example" {
  display_name = "support-agent"
  region       = "us-central1"

  encryption_spec {
    kms_key_name = "projects/my-project/locations/us-central1/keyRings/agents/cryptoKeys/agent-key"
  }
}
