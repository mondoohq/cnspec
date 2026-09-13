# The audit policy exists but turns audit logging off.
resource "alicloud_mongodb_instance" "prod" {
  engine_version      = "6.0"
  db_instance_class   = "dds.mongo.mid"
  db_instance_storage = 100
  vswitch_id          = alicloud_vswitch.db.id
  ssl_action          = "Open"
}

resource "alicloud_mongodb_audit_policy" "prod" {
  db_instance_id = alicloud_mongodb_instance.prod.id
  audit_status   = "disabled"
}
