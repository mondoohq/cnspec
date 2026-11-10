# Compliant: a public endpoint that requires identity domain authentication.
resource "oci_generative_ai_hosted_application" "example" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  display_name   = "agent"

  inbound_auth_config {
    inbound_auth_config_type = "IDCS_AUTH_CONFIG"
    idcs_config {
      domain_url = "https://idcs-example.identity.oraclecloud.com"
      scope      = "urn:opc:agent"
    }
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
