resource "alicloud_mongodb_instance" "shard" {
  count               = 2
  engine_version      = "6.0"
  db_instance_class   = "dds.mongo.mid"
  db_instance_storage = 100
  vswitch_id          = alicloud_vswitch.db.id
  ssl_action          = "Open"
}

resource "alicloud_mongodb_audit_policy" "shard" {
  count          = 2
  db_instance_id = alicloud_mongodb_instance.shard[count.index].id
  audit_status   = "enable"
}
