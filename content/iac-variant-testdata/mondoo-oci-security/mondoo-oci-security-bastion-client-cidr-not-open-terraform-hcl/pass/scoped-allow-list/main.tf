# Sessions can only be opened from the corporate egress range.
resource "oci_bastion_bastion" "ops" {
  bastion_type                 = "STANDARD"
  compartment_id               = var.compartment_ocid
  target_subnet_id             = oci_core_subnet.private.id
  name                         = "ops-bastion"
  client_cidr_block_allow_list = ["203.0.113.0/24"]
  max_session_ttl_in_seconds   = 3600
}
