# Non-compliant: deployment has no specification block at all.
resource "oci_apigateway_deployment" "no_spec" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  gateway_id     = "ocid1.apigateway.oc1.iad.aaaaaaaaexamplegateway"
  path_prefix    = "/open"
}
