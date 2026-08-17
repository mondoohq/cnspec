resource "databricks_ip_access_list" "open" {
  label        = "open"
  list_type    = "ALLOW"
  ip_addresses = ["203.0.113.0/24", "::/0"]
}
