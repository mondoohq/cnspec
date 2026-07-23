# Non-compliant: all TCP ports open to the internet (range spans 22 and 3389).
resource "oci_core_network_security_group_security_rule" "all" {
  network_security_group_id = oci_core_network_security_group.web.id
  direction                 = "INGRESS"
  protocol                  = "6"
  source                    = "0.0.0.0/0"
  source_type               = "CIDR_BLOCK"

  tcp_options {
    destination_port_range {
      min = 1
      max = 65535
    }
  }
}
