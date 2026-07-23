# Non-compliant: retention rule sets no time_rule_locked, so the rule is not locked.
resource "oci_objectstorage_bucket" "logs" {
  compartment_id = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  namespace      = "myobjectstorage"
  name           = "audit-logs"

  retention_rules {
    display_name = "unlocked-retention"
    duration {
      time_amount = 90
      time_unit   = "DAYS"
    }
  }
}
