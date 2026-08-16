# The condition names a destination but no source. An omitted source matches every
# address, so this rule allows the traffic from anywhere on the internet.
resource "oci_network_firewall_network_firewall_policy_security_rule" "allow_partner_api" {
  name                       = "allow-partner-api"
  network_firewall_policy_id = oci_network_firewall_network_firewall_policy.main.id
  action                     = "ALLOW"
  inspection                 = "INTRUSION_PREVENTION"

  condition {
    destination_address = ["app-tier"]
  }

  position {
    after_rule = "deny-all"
  }
}
