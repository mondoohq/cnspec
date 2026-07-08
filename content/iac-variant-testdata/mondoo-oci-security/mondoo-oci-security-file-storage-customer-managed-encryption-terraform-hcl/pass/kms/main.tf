# Compliant: file system encrypted with a customer-managed KMS key.
resource "oci_file_storage_file_system" "compliant" {
  availability_domain = "Uocm:PHX-AD-1"
  compartment_id      = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  display_name        = "shared-prod"
  kms_key_id          = "ocid1.key.oc1.phx.examplekeyvault.abcdefghijklmnop"
}
