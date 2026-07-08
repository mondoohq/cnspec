# Compliant: scoped grants, no manage all-resources in tenancy.
resource "oci_identity_policy" "compliant" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  name           = "network-admins"
  description    = "Scoped network administration"
  statements = [
    "Allow group NetworkAdmins to manage virtual-network-family in compartment Network",
    "Allow group Storage to read buckets in compartment Data"
  ]
}
