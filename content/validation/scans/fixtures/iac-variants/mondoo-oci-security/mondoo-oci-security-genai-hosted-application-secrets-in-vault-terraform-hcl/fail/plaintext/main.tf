# Non-compliant: a credential variable is stored as plaintext.
resource "oci_generative_ai_hosted_application" "example" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  display_name   = "agent"

  inbound_auth_config {
    inbound_auth_config_type = "IDCS_AUTH_CONFIG"
  }

  environment_variables {
    name  = "DB_PASSWORD"
    type  = "PLAINTEXT"
    value = "\"example-not-a-real-password\""
  }
}
