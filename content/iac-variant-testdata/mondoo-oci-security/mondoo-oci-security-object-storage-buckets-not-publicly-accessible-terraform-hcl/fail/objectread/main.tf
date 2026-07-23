# Non-compliant: anonymous read access to objects.
resource "oci_objectstorage_bucket" "data" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  namespace      = "examplenamespace"
  name           = "public-assets"
  access_type    = "ObjectRead"
}
