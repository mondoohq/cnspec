# Non-compliant: mTLS disabled and the access control list admits any address.
resource "oci_database_autonomous_database" "open_acl" {
  compartment_id              = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  db_name                     = "adbopenacl"
  cpu_core_count              = 1
  data_storage_size_in_tbs    = 1
  admin_password              = "BEstrO0ng_#11"
  db_workload                 = "OLTP"
  is_mtls_connection_required = false
  whitelisted_ips             = ["203.0.113.0/24", "0.0.0.0/0"]
}
