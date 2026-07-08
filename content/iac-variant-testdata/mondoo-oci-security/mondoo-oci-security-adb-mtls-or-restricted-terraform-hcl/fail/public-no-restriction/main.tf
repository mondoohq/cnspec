# Non-compliant: mTLS disabled and no network restriction of any kind.
resource "oci_database_autonomous_database" "open" {
  compartment_id              = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  db_name                     = "adbopen"
  cpu_core_count              = 1
  data_storage_size_in_tbs    = 1
  admin_password              = "BEstrO0ng_#11"
  db_workload                 = "OLTP"
  is_mtls_connection_required = false
}
