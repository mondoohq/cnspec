# Compliant: standby ACL lists only specific source networks.
resource "oci_database_autonomous_database" "example" {
  compartment_id           = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  db_name                  = "adbexample"
  cpu_core_count           = 1
  data_storage_size_in_tbs = 1
  admin_password           = "BEstr0ng_#12345"

  standby_whitelisted_ips = ["10.0.0.0/16", "192.168.1.0/24"]
}
