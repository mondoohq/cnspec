resource "digitalocean_dedicated_inference" "llm" {
  name                   = "private-inference"
  region                 = "atl1"
  enable_public_endpoint = true

  model_deployments {
    model_slug     = "mistral/mistral-7b-instruct-v3"
    model_provider = "hugging_face"

    accelerators {
      accelerator_slug = "gpu-mi300x1-192gb"
      scale            = 1
      type             = "prefill_decode"
    }
  }
}
