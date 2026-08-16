# Non-compliant: retention rule declares no duration block, so no minimum is set
# (an indefinite/legal-hold style rule that fails the existence guard).
resource "oci_objectstorage_bucket" "logs" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  namespace      = "myobjectstorage"
  name           = "audit-logs"

  retention_rules {
    display_name = "indefinite"
  }
}
