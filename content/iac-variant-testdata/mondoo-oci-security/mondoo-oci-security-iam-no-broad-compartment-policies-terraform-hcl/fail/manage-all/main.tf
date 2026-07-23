# Non-compliant: grants manage all-resources within a compartment.
resource "oci_identity_policy" "broad" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  name           = "compartment-admins"
  description    = "Over-broad compartment access"
  statements = [
    "Allow group Admins to manage all-resources in compartment Prod"
  ]
}
