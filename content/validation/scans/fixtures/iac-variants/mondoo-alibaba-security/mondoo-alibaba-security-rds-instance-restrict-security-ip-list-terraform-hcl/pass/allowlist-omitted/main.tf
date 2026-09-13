# With security_ips omitted the provider creates the instance with an allowlist
# of 127.0.0.1, which admits no remote client.
resource "alicloud_db_instance" "prod" {
  engine           = "MySQL"
  engine_version   = "8.0"
  instance_type    = "mysql.n2.medium.1"
  instance_storage = 100
  vswitch_id       = alicloud_vswitch.db.id
}
