# Non-compliant: legacy IMDSv1 endpoints remain enabled.
resource "oci_core_instance" "web" {
  compartment_id      = var.compartment_id
  availability_domain = var.availability_domain
  shape               = "VM.Standard.E4.Flex"
  display_name        = "web-01"

  source_details {
    source_type = "image"
    source_id   = var.image_id
  }

  create_vnic_details {
    subnet_id = var.subnet_id
  }

  instance_options {
    are_legacy_imds_endpoints_disabled = false
  }
}
