# Non-compliant: authentication block is present but type is empty.
resource "oci_apigateway_deployment" "empty_type" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  gateway_id     = "ocid1.apigateway.oc1.iad.aaaaaaaaexamplegateway"
  path_prefix    = "/v1"

  specification {
    request_policies {
      authentication {
        type                        = ""
        is_anonymous_access_allowed = true
      }
    }
  }
}
