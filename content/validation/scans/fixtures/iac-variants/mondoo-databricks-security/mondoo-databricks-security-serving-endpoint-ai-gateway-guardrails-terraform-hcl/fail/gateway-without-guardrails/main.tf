resource "databricks_model_serving" "chat" {
  name = "chat"
  ai_gateway {
    usage_tracking_config {
      enabled = true
    }
  }
}
