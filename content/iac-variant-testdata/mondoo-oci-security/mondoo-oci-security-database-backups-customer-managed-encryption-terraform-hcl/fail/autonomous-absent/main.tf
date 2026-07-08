# Non-compliant: no kms_key_id, so Oracle-managed encryption is used.
resource "oci_database_autonomous_database" "no_cmek" {
  compartment_id           = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  db_name                  = "adbdev01"
  cpu_core_count           = 1
  data_storage_size_in_tbs = 1
  admin_password           = "BEstrO0ng_#11"
  db_workload              = "OLTP"
}
