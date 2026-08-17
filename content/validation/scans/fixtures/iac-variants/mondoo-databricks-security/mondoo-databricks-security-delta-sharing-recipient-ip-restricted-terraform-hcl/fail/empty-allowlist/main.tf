resource "databricks_recipient" "partner" {
  name                = "partner"
  authentication_type = "TOKEN"
  ip_access_list {
    allowed_ip_addresses = []
  }
}
