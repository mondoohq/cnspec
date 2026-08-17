resource "databricks_ip_access_list" "deny" {
  label        = "deny-all-else"
  list_type    = "BLOCK"
  ip_addresses = ["0.0.0.0/0"]
}
