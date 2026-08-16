# Non-compliant: mutual_tls block present but is_verified_certificate_required omitted (defaults false).
resource "oci_apigateway_deployment" "mtls_default" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  gateway_id     = "ocid1.apigateway.oc1.iad.aaaaaaaaexamplegateway"
  path_prefix    = "/v1"

  specification {
    request_policies {
      mutual_tls {
        allowed_sans = ["client.example.com"]
      }
    }
  }
}
