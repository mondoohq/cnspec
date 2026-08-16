# Compliant (vacuous): bucket declares no retention_rules, so there is nothing
# to violate the minimum-duration requirement.
resource "oci_objectstorage_bucket" "scratch" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  namespace      = "myobjectstorage"
  name           = "scratch"
}
