# Non-compliant: public gateway with no network security group restriction.
resource "oci_apigateway_gateway" "public_open" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  endpoint_type  = "PUBLIC"
  subnet_id      = "ocid1.subnet.oc1.iad.aaaaaaaaexamplesubnet"
  display_name   = "public-gateway"
}
