# Compliant: deployment configures JWT authentication in request_policies.
resource "oci_apigateway_deployment" "compliant" {
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

        public_keys {
          type = "REMOTE_JWKS"
          uri  = "https://identity.example.com/.well-known/jwks.json"
        }
      }
    }
  }
}
