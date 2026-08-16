# A public load balancer with no NSG is filtered only by the subnet security list, which
# is shared with everything else in that subnet and written for the broadest of them. A
# layer 4 load balancer has no listener authentication to fall back on.
resource "oci_network_load_balancer_network_load_balancer" "public" {
  compartment_id = var.compartment_ocid
  display_name   = "public-nlb"
  subnet_id      = oci_core_subnet.public.id
  is_private     = false
}
