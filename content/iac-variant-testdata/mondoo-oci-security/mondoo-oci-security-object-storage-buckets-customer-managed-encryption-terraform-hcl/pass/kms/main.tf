# Compliant: bucket encrypted with a customer-managed KMS key.
resource "oci_objectstorage_bucket" "data" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  namespace      = "examplenamespace"
  name           = "sensitive-data"
  kms_key_id     = "ocid1.key.oc1.us-phoenix-1.examplekey"
}
