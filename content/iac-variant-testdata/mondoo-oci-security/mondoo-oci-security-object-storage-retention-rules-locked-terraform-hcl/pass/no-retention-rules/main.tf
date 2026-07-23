# Compliant (vacuous): no retention_rules blocks to require a lock time.
resource "oci_objectstorage_bucket" "scratch" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  namespace      = "myobjectstorage"
  name           = "scratch"
}
