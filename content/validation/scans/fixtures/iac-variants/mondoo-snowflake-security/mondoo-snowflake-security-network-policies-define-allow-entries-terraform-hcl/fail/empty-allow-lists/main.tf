# Both allow lists present but empty, which reads as "configured" while
# constraining nothing.
resource "snowflake_network_policy" "corporate" {
  name                      = "CORPORATE_ACCESS"
  allowed_ip_list           = []
  allowed_network_rule_list = []
  blocked_ip_list           = ["203.0.113.99"]
}
