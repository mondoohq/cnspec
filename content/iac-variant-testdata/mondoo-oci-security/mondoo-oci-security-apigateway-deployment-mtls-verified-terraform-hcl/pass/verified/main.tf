# Compliant: mutual TLS requires a verified client certificate.
resource "oci_apigateway_deployment" "verified" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  gateway_id     = "ocid1.apigateway.oc1.iad.aaaaaaaaexamplegateway"
  path_prefix    = "/v1"

  specification {
    request_policies {
      mutual_tls {
        is_verified_certificate_required = true
        allowed_sans                     = ["client.example.com"]
      }
    }
  }
}
