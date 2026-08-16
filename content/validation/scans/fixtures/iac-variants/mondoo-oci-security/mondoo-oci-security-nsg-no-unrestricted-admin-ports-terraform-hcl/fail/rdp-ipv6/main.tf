# Non-compliant: RDP (3389) open to the internet over IPv6.
resource "oci_core_network_security_group_security_rule" "rdp" {
  network_security_group_id = oci_core_network_security_group.web.id
  direction                 = "INGRESS"
  protocol                  = "6"
  source                    = "::/0"
  source_type               = "CIDR_BLOCK"

  tcp_options {
    destination_port_range {
      min = 3389
      max = 3389
    }
  }
}
