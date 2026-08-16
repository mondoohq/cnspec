# Compliant: gateway is private, not exposed publicly.
resource "oci_apigateway_gateway" "private" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  endpoint_type  = "PRIVATE"
  subnet_id      = "ocid1.subnet.oc1.iad.aaaaaaaaexamplesubnet"
  display_name   = "internal-gateway"
}
