# Non-compliant: bucket relies on Oracle-managed encryption (no kms_key_id).
resource "oci_objectstorage_bucket" "data" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  namespace      = "examplenamespace"
  name           = "sensitive-data"
}
