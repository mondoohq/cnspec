# Compliant: every dynamic authentication server rejects anonymous requests.
resource "oci_apigateway_deployment" "dynamic" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  gateway_id     = "ocid1.apigateway.oc1.iad.aaaaaaaaexamplegateway"
  path_prefix    = "/v1"

  specification {
    request_policies {
      dynamic_authentication {
        selection_source {
          type     = "SINGLE"
          selector = "request.headers[X-Tenant]"
        }

        authentication_servers {
          key {
            type       = "ANY_OF"
            name       = "tenants"
            values     = ["a", "b"]
            is_default = true
          }

          authentication_server_detail {
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
  }
}
