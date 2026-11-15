# Non-compliant: private endpoint that also allows public access, which publishes
# a public copy of the management URLs.
resource "oci_database_autonomous_database" "pe_public" {
  compartment_id           = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  db_name                  = "adbpepub"
  cpu_core_count           = 1
  data_storage_size_in_tbs = 1
  admin_password           = "BEstr0ng_#12345"
  subnet_id                = "ocid1.subnet.oc1.iad.aaaaaaaaexamplesubnet"
  whitelisted_ips          = ["203.0.113.0/24"]
}
