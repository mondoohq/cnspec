resource "digitalocean_agent_platform_knowledge_base" "docs" {
  name                 = "product-docs"
  project_id           = "00000000-0000-0000-0000-000000000000"
  region               = "tor1"
  embedding_model_uuid = "00000000-0000-0000-0000-000000000000"
  is_public            = false

  datasources {
    spaces_data_source {
      bucket_name = "product-docs"
      region      = "tor1"
    }
  }
}
