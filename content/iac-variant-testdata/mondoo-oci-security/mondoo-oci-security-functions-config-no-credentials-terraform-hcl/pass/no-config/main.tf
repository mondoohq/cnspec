# Compliant: application defines no config map, so there are no secrets to leak.
resource "oci_functions_application" "compliant" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  display_name   = "image-resizer"
  subnet_ids     = ["ocid1.subnet.oc1.phx.examplesubnet.abcdefghijklmnop"]
}
