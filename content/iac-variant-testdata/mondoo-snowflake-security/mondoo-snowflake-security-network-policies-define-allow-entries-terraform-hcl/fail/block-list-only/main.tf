# A policy with only a block list is deny-by-exception: every address that is
# not explicitly listed still reaches the account.
resource "snowflake_network_policy" "corporate" {
  name            = "CORPORATE_ACCESS"
  blocked_ip_list = ["203.0.113.99"]
  comment         = "Blocks one bad actor and allows everything else"
}
