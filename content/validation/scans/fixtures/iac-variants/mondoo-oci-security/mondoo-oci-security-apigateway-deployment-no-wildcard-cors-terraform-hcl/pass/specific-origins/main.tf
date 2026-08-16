# Compliant: CORS allows only specific origins.
resource "oci_apigateway_deployment" "cors_ok" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  gateway_id     = "ocid1.apigateway.oc1.iad.aaaaaaaaexamplegateway"
  path_prefix    = "/v1"

  specification {
    request_policies {
      cors {
        allowed_origins = ["https://app.example.com", "https://admin.example.com"]
        allowed_methods = ["GET", "POST"]
      }
    }
  }
}
