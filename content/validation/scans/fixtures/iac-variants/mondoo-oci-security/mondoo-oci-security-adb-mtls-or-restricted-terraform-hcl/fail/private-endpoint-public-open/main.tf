# Non-compliant: private endpoint with public access allowed from any address.
resource "oci_database_autonomous_database" "pe_public" {
  compartment_id              = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  db_name                     = "adbpepub"
  cpu_core_count              = 1
  data_storage_size_in_tbs    = 1
  admin_password              = "BEstrO0ng_#11"
  db_workload                 = "OLTP"
  is_mtls_connection_required = false
  subnet_id                   = "ocid1.subnet.oc1.iad.aaaaaaaaexamplesubnet"
  whitelisted_ips             = ["0.0.0.0/0"]
}
