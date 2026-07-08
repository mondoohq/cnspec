# Compliant: autonomous database uses a customer-managed KMS key.
resource "oci_database_autonomous_database" "compliant" {
  compartment_id           = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  db_name                  = "adbprod01"
  cpu_core_count           = 2
  data_storage_size_in_tbs = 1
  admin_password           = "BEstrO0ng_#11"
  db_workload              = "OLTP"
  kms_key_id               = "ocid1.key.oc1.iad.examplekeyvault.abcdefghijklmnop"
  vault_id                 = "ocid1.vault.oc1.iad.examplevault.abcdefghijklmnop"
}
