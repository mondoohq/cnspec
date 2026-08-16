# Compliant: object events are emitted for the bucket.
resource "oci_objectstorage_bucket" "data" {
  compartment_id        = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  namespace             = "examplenamespace"
  name                  = "audited-data"
  object_events_enabled = true
}
