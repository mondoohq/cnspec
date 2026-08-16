# Non-compliant: DB database block has no kms_key_id (Oracle-managed encryption).
resource "oci_database_database" "no_cmek" {
  db_home_id = "ocid1.dbhome.oc1.iad.exampledbhome.abcdefghijklmnop"
  source     = "NONE"

  database {
    admin_password = "BEstrO0ng_#11"
    db_name        = "dev01"
    pdb_name       = "pdb01"
  }
}
