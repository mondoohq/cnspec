# Non-compliant: request_policies is present but declares no authentication.
resource "oci_apigateway_deployment" "no_auth" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  gateway_id     = "ocid1.apigateway.oc1.iad.aaaaaaaaexamplegateway"
  path_prefix    = "/v1"

  specification {
    request_policies {
      cors {
        allowed_origins = ["https://app.example.com"]
        allowed_methods = ["GET"]
      }
    }
  }
}
