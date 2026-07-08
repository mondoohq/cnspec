# Non-compliant: kms_key_id explicitly set to an empty string.
resource "oci_database_autonomous_database" "empty_cmek" {
  compartment_id           = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  db_name                  = "adbdev02"
  cpu_core_count           = 1
  data_storage_size_in_tbs = 1
  admin_password           = "BEstrO0ng_#11"
  db_workload              = "OLTP"
  kms_key_id               = ""
}
