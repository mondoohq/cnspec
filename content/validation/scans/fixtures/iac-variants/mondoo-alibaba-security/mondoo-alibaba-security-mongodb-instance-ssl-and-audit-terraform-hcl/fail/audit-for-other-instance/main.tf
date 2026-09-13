# Only one of the two instances has an audit policy pointing at it.
resource "alicloud_mongodb_instance" "prod" {
  engine_version      = "6.0"
  db_instance_class   = "dds.mongo.mid"
  db_instance_storage = 100
  vswitch_id          = alicloud_vswitch.db.id
  ssl_action          = "Open"
}

resource "alicloud_mongodb_instance" "reporting" {
  engine_version      = "6.0"
  db_instance_class   = "dds.mongo.mid"
  db_instance_storage = 100
  vswitch_id          = alicloud_vswitch.db.id
  ssl_action          = "Open"
}

resource "alicloud_mongodb_audit_policy" "prod" {
  db_instance_id = alicloud_mongodb_instance.prod.id
  audit_status   = "enable"
}
