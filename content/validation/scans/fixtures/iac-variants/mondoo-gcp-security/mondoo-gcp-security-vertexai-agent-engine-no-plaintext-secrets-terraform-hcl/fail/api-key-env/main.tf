# Non-compliant: an API key is passed as a plain environment variable.
resource "google_vertex_ai_reasoning_engine" "fail_example" {
  display_name = "support-agent"
  region       = "us-central1"

  spec {
    deployment_spec {
      env {
        name  = "OPENAI_API_KEY"
        value = "example-value"
      }
    }
  }
}
