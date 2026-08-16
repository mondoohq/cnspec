# The allow rule names the address lists traffic is expected from.
resource "oci_network_firewall_network_firewall_policy_security_rule" "allow_partner_api" {
  name                       = "allow-partner-api"
  network_firewall_policy_id = oci_network_firewall_network_firewall_policy.main.id
  action                     = "ALLOW"
  inspection                 = "INTRUSION_PREVENTION"

  condition {
    source_address      = ["corporate-ranges"]
    destination_address = ["app-tier"]
  }

  position {
    after_rule = "deny-all"
  }
}
