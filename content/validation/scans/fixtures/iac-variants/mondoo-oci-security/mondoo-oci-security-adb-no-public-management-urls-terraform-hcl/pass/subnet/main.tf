# Compliant: database is deployed into a private subnet.
resource "oci_database_autonomous_database" "private_subnet" {
  compartment_id           = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  db_name                  = "adbnet"
  cpu_core_count           = 1
  data_storage_size_in_tbs = 1
  admin_password           = "BEstrO0ng_#11"
  db_workload              = "OLTP"
  subnet_id                = "ocid1.subnet.oc1.iad.aaaaaaaaexamplesubnet"
}
