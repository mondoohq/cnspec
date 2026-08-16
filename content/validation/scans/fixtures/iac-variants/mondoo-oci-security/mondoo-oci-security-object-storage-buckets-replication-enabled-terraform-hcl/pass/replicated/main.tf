# Compliant: every bucket has a matching replication policy.
resource "oci_objectstorage_bucket" "data" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  namespace      = "examplenamespace"
  name           = "primary-data"
}

resource "oci_objectstorage_replication_policy" "data" {
  bucket             = oci_objectstorage_bucket.data.name
  namespace          = "examplenamespace"
  name               = "to-ashburn"
  destination_region_name    = "us-ashburn-1"
  destination_bucket_name    = "primary-data-replica"
}
