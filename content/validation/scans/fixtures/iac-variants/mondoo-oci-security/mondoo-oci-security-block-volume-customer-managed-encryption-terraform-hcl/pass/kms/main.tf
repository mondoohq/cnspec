# Compliant: block volume uses a customer-managed KMS key.
resource "oci_core_volume" "compliant" {
  compartment_id      = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  availability_domain = "Uocm:PHX-AD-1"
  display_name        = "data-volume"
  size_in_gbs         = 50
  kms_key_id          = "ocid1.key.oc1.iad.examplekeyvault.abcdefghijklmnop"
}
