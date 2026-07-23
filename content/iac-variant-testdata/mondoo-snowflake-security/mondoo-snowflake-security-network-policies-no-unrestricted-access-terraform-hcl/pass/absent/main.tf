resource "snowflake_network_policy" "corporate" {
  name            = "CORPORATE_POLICY"
  blocked_ip_list = ["203.0.113.0/24"]
}
