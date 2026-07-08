# Non-compliant: assign_public_ip is omitted (defaults to true on a public
# subnet) and no network security group is attached.
resource "oci_core_instance" "web" {
  availability_domain = "Uocm:PHX-AD-1"
  compartment_id      = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  shape               = "VM.Standard.E4.Flex"

  create_vnic_details {
    subnet_id = "ocid1.subnet.oc1.iad.aaaaaaaaexamplesubnet"
  }
}
