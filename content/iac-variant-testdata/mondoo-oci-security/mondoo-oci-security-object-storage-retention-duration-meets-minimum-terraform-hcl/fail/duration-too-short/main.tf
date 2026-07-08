# Non-compliant: retention duration of 7 DAYS is below the 30-day minimum.
resource "oci_objectstorage_bucket" "logs" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  namespace      = "myobjectstorage"
  name           = "audit-logs"

  retention_rules {
    display_name = "too-short"
    duration {
      time_amount = 7
      time_unit   = "DAYS"
    }
  }
}
