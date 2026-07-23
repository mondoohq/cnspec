# Compliant: retention duration of 30 DAYS meets the minimum.
resource "oci_objectstorage_bucket" "logs" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  namespace      = "myobjectstorage"
  name           = "audit-logs"

  retention_rules {
    display_name = "minimum-retention"
    duration {
      time_amount = 30
      time_unit   = "DAYS"
    }
  }
}
