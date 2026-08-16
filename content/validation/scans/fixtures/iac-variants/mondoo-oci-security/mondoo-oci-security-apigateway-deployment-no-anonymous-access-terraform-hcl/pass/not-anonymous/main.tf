# Compliant: anonymous access is explicitly disallowed.
resource "oci_apigateway_deployment" "no_anon" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  gateway_id     = "ocid1.apigateway.oc1.iad.aaaaaaaaexamplegateway"
  path_prefix    = "/v1"

  specification {
    request_policies {
      authentication {
        type                        = "JWT_AUTHENTICATION"
        token_header                = "Authorization"
        issuers                     = ["https://identity.example.com"]
        audiences                   = ["api://default"]
        is_anonymous_access_allowed = false
      }
    }
  }
}
