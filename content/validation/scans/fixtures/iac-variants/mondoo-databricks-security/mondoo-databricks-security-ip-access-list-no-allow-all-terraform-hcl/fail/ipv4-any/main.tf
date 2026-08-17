resource "databricks_ip_access_list" "open" {
  label        = "open"
  list_type    = "ALLOW"
  ip_addresses = ["0.0.0.0/0"]
}
