resource "snowflake_network_policy" "corporate" {
  name            = "CORPORATE_ACCESS"
  allowed_ip_list = ["203.0.113.0/24", "198.51.100.14"]
  blocked_ip_list = ["203.0.113.99"]
  comment         = "Office egress ranges"
}
