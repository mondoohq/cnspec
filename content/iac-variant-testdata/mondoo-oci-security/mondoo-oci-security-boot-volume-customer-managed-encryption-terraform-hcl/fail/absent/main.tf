# Non-compliant: no kms_key_id, so the boot volume uses Oracle-managed encryption.
resource "oci_core_boot_volume" "no_cmek" {
  compartment_id      = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  availability_domain = "Uocm:PHX-AD-1"
  display_name        = "instance-boot-volume"
  source_details {
    type = "bootVolume"
    id   = "ocid1.bootvolume.oc1.phx.aaaaaaaaexamplesource"
  }
}
