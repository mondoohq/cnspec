resource "snowflake_network_policy" "corporate" {
  name            = "CORPORATE_POLICY"
  allowed_ip_list = ["192.168.1.0/24", "10.0.0.0/8"]
}
