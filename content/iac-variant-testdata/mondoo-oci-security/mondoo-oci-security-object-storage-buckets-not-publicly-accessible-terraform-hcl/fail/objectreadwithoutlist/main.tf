# Non-compliant: anonymous read access without listing.
resource "oci_objectstorage_bucket" "data" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  namespace      = "examplenamespace"
  name           = "public-assets"
  access_type    = "ObjectReadWithoutList"
}
