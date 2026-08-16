# Non-compliant: versioning disabled.
resource "oci_objectstorage_bucket" "data" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  namespace      = "examplenamespace"
  name           = "versioned-data"
  versioning     = "Disabled"
}
