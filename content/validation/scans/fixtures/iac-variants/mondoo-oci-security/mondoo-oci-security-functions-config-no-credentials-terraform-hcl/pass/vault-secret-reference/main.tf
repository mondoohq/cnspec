# Compliant: the config names a Vault secret OCID rather than holding the secret.
resource "oci_functions_application" "vault_ref" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  display_name   = "order-processor"
  subnet_ids     = ["ocid1.subnet.oc1.phx.examplesubnet.abcdefghijklmnop"]

  config = {
    DB_PASSWORD_SECRET_OCID = "ocid1.vaultsecret.oc1.phx.examplesecret"
    LOG_LEVEL               = "info"
  }
}
