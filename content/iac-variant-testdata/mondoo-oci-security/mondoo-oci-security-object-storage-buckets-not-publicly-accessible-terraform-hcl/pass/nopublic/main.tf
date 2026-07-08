# Compliant: access type explicitly set to no public access.
resource "oci_objectstorage_bucket" "data" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  namespace      = "examplenamespace"
  name           = "private-data"
  access_type    = "NoPublicAccess"
}
