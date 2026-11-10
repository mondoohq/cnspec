# Compliant: no plain environment variables at all.
resource "google_vertex_ai_reasoning_engine" "pass_example" {
  display_name = "support-agent"
  region       = "us-central1"
}
