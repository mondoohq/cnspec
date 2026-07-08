# Compliant: public gateway restricted by network security groups.
resource "oci_apigateway_gateway" "public_nsg" {
  compartment_id             = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  endpoint_type              = "PUBLIC"
  subnet_id                  = "ocid1.subnet.oc1.iad.aaaaaaaaexamplesubnet"
  display_name               = "public-gateway"
  network_security_group_ids = ["ocid1.networksecuritygroup.oc1.iad.aaaaaaaaexamplensg"]
}
