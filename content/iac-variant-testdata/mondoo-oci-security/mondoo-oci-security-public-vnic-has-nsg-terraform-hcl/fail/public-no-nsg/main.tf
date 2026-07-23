# Non-compliant: VNIC is assigned a public IP with no network security group.
resource "oci_core_instance" "web" {
  availability_domain = "Uocm:PHX-AD-1"
  compartment_id      = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  shape               = "VM.Standard.E4.Flex"

  create_vnic_details {
    subnet_id        = "ocid1.subnet.oc1.iad.aaaaaaaaexamplesubnet"
    assign_public_ip = true
  }
}
