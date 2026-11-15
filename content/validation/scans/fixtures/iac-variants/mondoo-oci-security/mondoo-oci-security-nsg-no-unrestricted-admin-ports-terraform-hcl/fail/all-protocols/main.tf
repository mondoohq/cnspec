# Non-compliant: a rule for all protocols from the internet covers SSH and RDP.
resource "oci_core_network_security_group_security_rule" "all" {
  network_security_group_id = "ocid1.networksecuritygroup.oc1.iad.aaaaaaaaexamplensg"
  direction                 = "INGRESS"
  protocol                  = "all"
  source                    = "0.0.0.0/0"
  source_type               = "CIDR_BLOCK"
}
