# Non-compliant: a public endpoint that accepts unauthenticated requests.
resource "oci_generative_ai_hosted_application" "example" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  display_name   = "agent"

  inbound_auth_config {
    inbound_auth_config_type = "NO_AUTH_CONFIG"
  }

  networking_config {
    inbound_networking_config {
      endpoint_mode = "PUBLIC"
    }
    outbound_networking_config {
      network_mode = "MANAGED"
    }
  }
}
