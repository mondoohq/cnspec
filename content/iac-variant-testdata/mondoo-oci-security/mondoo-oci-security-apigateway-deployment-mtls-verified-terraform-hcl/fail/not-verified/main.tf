# Non-compliant: mutual TLS present but verified certificate is not required.
resource "oci_apigateway_deployment" "not_verified" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  gateway_id     = "ocid1.apigateway.oc1.iad.aaaaaaaaexamplegateway"
  path_prefix    = "/v1"

  specification {
    request_policies {
      mutual_tls {
        is_verified_certificate_required = false
      }
    }
  }
}
