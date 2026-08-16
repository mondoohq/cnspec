# Compliant: retention rule declares a lock time.
resource "oci_objectstorage_bucket" "logs" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  namespace      = "myobjectstorage"
  name           = "audit-logs"

  retention_rules {
    display_name = "locked-retention"
    duration {
      time_amount = 90
      time_unit   = "DAYS"
    }
    time_rule_locked = "2025-01-01T00:00:00.000Z"
  }
}
