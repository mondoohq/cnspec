resource "snowflake_network_policy" "corporate" {
  name            = "CORPORATE_POLICY"
  allowed_ip_list = ["0.0.0.0/0"]
}
