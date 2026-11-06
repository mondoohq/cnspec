resource "digitalocean_agent_platform_agent" "support" {
  name        = "support-agent"
  instruction = "Answer questions about the product documentation."
  model_uuid  = "00000000-0000-0000-0000-000000000000"
  project_id  = "00000000-0000-0000-0000-000000000000"
  region      = "tor1"

  deployment {
    visibility = "VISIBILITY_PRIVATE"
  }
}
