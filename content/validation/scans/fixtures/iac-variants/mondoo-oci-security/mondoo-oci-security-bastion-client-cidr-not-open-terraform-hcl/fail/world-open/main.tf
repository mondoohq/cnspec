# The bastion is the one path into the private subnet, and this publishes it to the
# entire internet.
resource "oci_bastion_bastion" "ops" {
  bastion_type                 = "STANDARD"
  compartment_id               = var.compartment_ocid
  target_subnet_id             = oci_core_subnet.private.id
  name                         = "ops-bastion"
  client_cidr_block_allow_list = ["0.0.0.0/0"]
  max_session_ttl_in_seconds   = 3600
}
