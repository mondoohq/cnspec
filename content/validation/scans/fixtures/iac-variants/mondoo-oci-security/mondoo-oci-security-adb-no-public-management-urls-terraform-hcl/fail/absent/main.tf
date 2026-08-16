# Non-compliant: no subnet_id, so the database has public management URLs.
resource "oci_database_autonomous_database" "public_urls" {
  compartment_id           = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  db_name                  = "adbpub"
  cpu_core_count           = 1
  data_storage_size_in_tbs = 1
  admin_password           = "BEstrO0ng_#11"
  db_workload              = "OLTP"
}
