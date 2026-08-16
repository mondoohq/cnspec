# Compliant: boot volume uses a customer-managed KMS key.
resource "oci_core_boot_volume" "compliant" {
  compartment_id      = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  availability_domain = "Uocm:PHX-AD-1"
  display_name        = "instance-boot-volume"
  source_details {
    type = "bootVolume"
    id   = "ocid1.bootvolume.oc1.phx.aaaaaaaaexamplesource"
  }
  kms_key_id = "ocid1.key.oc1.iad.examplekeyvault.abcdefghijklmnop"
}
