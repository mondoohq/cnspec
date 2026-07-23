# Compliant (vacuously): a Standard-tier bucket is not in scope for the Archive check.
resource "oci_objectstorage_bucket" "standard" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  namespace      = "examplenamespace"
  name           = "hot-data"
  storage_tier   = "Standard"
  versioning     = "Disabled"
}
