# Compliant: the credential is a Secret Manager reference; plain env holds no secret-like names.
resource "google_vertex_ai_reasoning_engine" "pass_example" {
  display_name = "support-agent"
  region       = "us-central1"

  spec {
    deployment_spec {
      env {
        name  = "LOG_LEVEL"
        value = "info"
      }

      secret_env {
        name = "MODEL_PROVIDER_API_KEY"

        secret_ref {
          secret  = "model-provider-key"
          version = "latest"
        }
      }
    }
  }
}
