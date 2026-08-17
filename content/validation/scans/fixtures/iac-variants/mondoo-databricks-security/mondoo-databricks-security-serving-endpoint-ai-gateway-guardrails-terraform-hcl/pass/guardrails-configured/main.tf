resource "databricks_model_serving" "chat" {
  name = "chat"
  ai_gateway {
    guardrails {
      input {
        safety = true
      }
      output {
        safety = true
      }
    }
  }
}
