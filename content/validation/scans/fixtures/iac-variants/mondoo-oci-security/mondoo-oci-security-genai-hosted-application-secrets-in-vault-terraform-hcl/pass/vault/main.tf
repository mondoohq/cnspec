# Compliant: the credential variable reads a vault secret; the plain variable is not a credential.
resource "oci_generative_ai_hosted_application" "example" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  display_name   = "agent"

  inbound_auth_config {
    inbound_auth_config_type = "IDCS_AUTH_CONFIG"
  }

  environment_variables {
    name  = "MODEL_API_KEY"
    type  = "VAULT"
    value = "{\"vaultId\":\"ocid1.vault.oc1..aaaaaaaaexample\",\"key\":\"model-api-key\"}"
  }

  environment_variables {
    name  = "LOG_LEVEL"
    type  = "PLAINTEXT"
    value = "\"info\""
  }
}
