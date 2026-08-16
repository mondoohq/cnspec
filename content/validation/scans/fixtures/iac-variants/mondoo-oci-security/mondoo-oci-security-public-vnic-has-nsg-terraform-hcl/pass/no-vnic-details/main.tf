# Compliant (vacuous): instance declares no create_vnic_details block.
resource "oci_core_instance" "app" {
  availability_domain = "Uocm:PHX-AD-1"
  compartment_id      = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  shape               = "VM.Standard.E4.Flex"
  subnet_id           = "ocid1.subnet.oc1.iad.aaaaaaaaexamplesubnet"
}
