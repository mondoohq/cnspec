# Non-compliant: internet TCP ingress with no port restriction at all.
resource "oci_core_network_security_group_security_rule" "all" {
  network_security_group_id = oci_core_network_security_group.web.id
  direction                 = "INGRESS"
  protocol                  = "6"
  source                    = "0.0.0.0/0"
  source_type               = "CIDR_BLOCK"
}
