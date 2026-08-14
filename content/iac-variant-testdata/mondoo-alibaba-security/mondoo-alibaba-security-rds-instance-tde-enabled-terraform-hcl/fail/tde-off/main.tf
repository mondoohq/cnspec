# Without TDE the data files and backups sit unencrypted at rest.
resource "alicloud_db_instance" "prod" {
  engine           = "MySQL"
  engine_version   = "8.0"
  instance_type    = "mysql.n2.medium.1"
  instance_storage = 100
  vswitch_id       = alicloud_vswitch.db.id
  tde_status       = "Disabled"
}
