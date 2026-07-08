# Non-compliant: Archive bucket with versioning disabled.
resource "oci_objectstorage_bucket" "archive" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  namespace      = "examplenamespace"
  name           = "long-term-archive"
  storage_tier   = "Archive"
  versioning     = "Disabled"
}
