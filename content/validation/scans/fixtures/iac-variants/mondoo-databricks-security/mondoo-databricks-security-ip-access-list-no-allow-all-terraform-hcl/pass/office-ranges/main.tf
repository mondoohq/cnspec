resource "databricks_ip_access_list" "office" {
  label        = "office"
  list_type    = "ALLOW"
  ip_addresses = ["203.0.113.0/24", "198.51.100.7/32"]
}
