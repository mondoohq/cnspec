# Non-compliant: db_home/database has no kms_key_id (Oracle-managed encryption).
resource "oci_database_db_system" "no_cmek" {
  availability_domain = "Uocm:PHX-AD-1"
  compartment_id      = "ocid1.compartment.oc1..aaaaaaaaexamplecompartment"
  database_edition    = "ENTERPRISE_EDITION"
  shape               = "VM.Standard2.2"
  subnet_id           = "ocid1.subnet.oc1.phx.examplesubnet.abcdefghijklmnop"
  cpu_core_count      = 2
  hostname            = "dbdev"
  ssh_public_keys     = ["ssh-rsa AAAAB3NzaC1yc2EAAAADAQABexamplekey user@host"]

  db_home {
    db_version = "19.0.0.0"

    database {
      admin_password = "BEstrO0ng_#11"
      db_name        = "dev"
    }
  }
}
