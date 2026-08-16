# Non-compliant: standby ACL admits any IPv4 address.
resource "oci_database_autonomous_database" "example" {
  compartment_id           = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  db_name                  = "adbexample"
  cpu_core_count           = 1
  data_storage_size_in_tbs = 1
  admin_password           = "BEstr0ng_#12345"

  standby_whitelisted_ips = ["10.0.0.0/16", "0.0.0.0/0"]
}
