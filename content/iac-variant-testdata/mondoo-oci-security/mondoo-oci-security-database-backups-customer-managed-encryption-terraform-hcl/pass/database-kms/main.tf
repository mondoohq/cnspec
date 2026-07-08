# Compliant: DB database resource with a customer-managed KMS key in the database block.
resource "oci_database_database" "compliant" {
  db_home_id = "ocid1.dbhome.oc1.iad.exampledbhome.abcdefghijklmnop"
  source     = "NONE"

  database {
    admin_password = "BEstrO0ng_#11"
    db_name        = "prod01"
    pdb_name       = "pdb01"
    kms_key_id     = "ocid1.key.oc1.iad.examplekeyvault.abcdefghijklmnop"
    vault_id       = "ocid1.vault.oc1.iad.examplevault.abcdefghijklmnop"
  }
}
