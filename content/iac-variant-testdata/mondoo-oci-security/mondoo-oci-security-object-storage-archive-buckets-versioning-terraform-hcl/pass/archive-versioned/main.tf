# Compliant: Archive bucket with versioning enabled.
resource "oci_objectstorage_bucket" "archive" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  namespace      = "examplenamespace"
  name           = "long-term-archive"
  storage_tier   = "Archive"
  versioning     = "Enabled"
}
