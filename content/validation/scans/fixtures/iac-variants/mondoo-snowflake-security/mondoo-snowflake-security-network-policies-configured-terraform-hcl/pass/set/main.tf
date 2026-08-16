resource "snowflake_account_parameter" "network_policy" {
  key   = "NETWORK_POLICY"
  value = "corporate_network_policy"
}
