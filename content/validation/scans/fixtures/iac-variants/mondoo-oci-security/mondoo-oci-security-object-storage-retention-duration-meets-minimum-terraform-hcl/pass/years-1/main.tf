# Compliant: retention duration of 1 YEARS meets the minimum.
resource "oci_objectstorage_bucket" "archive" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  namespace      = "myobjectstorage"
  name           = "compliance-archive"

  retention_rules {
    display_name = "one-year"
    duration {
      time_amount = 1
      time_unit   = "YEARS"
    }
  }
}
