# Compliant: internet ingress limited to an application port range above 3389.
resource "oci_core_network_security_group_security_rule" "app" {
  network_security_group_id = oci_core_network_security_group.web.id
  direction                 = "INGRESS"
  protocol                  = "6"
  source                    = "0.0.0.0/0"
  source_type               = "CIDR_BLOCK"

  tcp_options {
    destination_port_range {
      min = 8080
      max = 8090
    }
  }
}
