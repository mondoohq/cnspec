# Non-compliant: freeform_tags is present but empty.
resource "oci_core_instance" "web" {
  compartment_id      = var.compartment_id
  availability_domain = var.availability_domain
  shape               = "VM.Standard.E4.Flex"
  display_name        = "web-01"

  freeform_tags = {}

  source_details {
    source_type = "image"
    source_id   = var.image_id
  }

  create_vnic_details {
    subnet_id = var.subnet_id
  }
}
