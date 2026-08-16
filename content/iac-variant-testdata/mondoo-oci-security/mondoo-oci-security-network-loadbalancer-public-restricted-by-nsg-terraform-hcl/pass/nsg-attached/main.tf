# The public load balancer is gated by a network security group scoped to its listeners.
resource "oci_network_load_balancer_network_load_balancer" "public" {
  compartment_id             = var.compartment_ocid
  display_name               = "public-nlb"
  subnet_id                  = oci_core_subnet.public.id
  is_private                 = false
  network_security_group_ids = [oci_core_network_security_group.nlb.id]
}

# A private load balancer has no internet path, so it needs no NSG to pass.
resource "oci_network_load_balancer_network_load_balancer" "internal" {
  compartment_id = var.compartment_ocid
  display_name   = "internal-nlb"
  subnet_id      = oci_core_subnet.private.id
  is_private     = true
}
