resource "databricks_recipient" "partner" {
  name                = "partner"
  authentication_type = "TOKEN"
}
