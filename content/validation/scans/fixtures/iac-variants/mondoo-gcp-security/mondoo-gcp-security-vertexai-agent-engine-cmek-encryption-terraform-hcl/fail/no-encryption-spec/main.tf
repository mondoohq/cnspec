# Non-compliant: no encryption_spec, so a Google-managed key is used.
resource "google_vertex_ai_reasoning_engine" "fail_example" {
  display_name = "support-agent"
  region       = "us-central1"
}
