# Non-compliant: CORS allows any origin via a wildcard.
resource "oci_apigateway_deployment" "cors_wildcard" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  gateway_id     = "ocid1.apigateway.oc1.iad.aaaaaaaaexamplegateway"
  path_prefix    = "/v1"

  specification {
    request_policies {
      cors {
        allowed_origins = ["*"]
        allowed_methods = ["GET", "POST"]
      }
    }
  }
}
