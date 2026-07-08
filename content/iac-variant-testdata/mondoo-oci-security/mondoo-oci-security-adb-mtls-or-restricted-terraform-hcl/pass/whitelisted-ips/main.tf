# Compliant: access is restricted to an IP allow list.
resource "oci_database_autonomous_database" "whitelist" {
  compartment_id              = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  db_name                     = "adbwl"
  cpu_core_count              = 1
  data_storage_size_in_tbs    = 1
  admin_password              = "BEstrO0ng_#11"
  db_workload                 = "OLTP"
  is_mtls_connection_required = false
  whitelisted_ips             = ["203.0.113.10", "203.0.113.0/24"]
}
