# Client connections, including credentials and query payloads, are unencrypted
# in transit.
resource "alicloud_mongodb_instance" "prod" {
  engine_version      = "6.0"
  db_instance_class   = "dds.mongo.mid"
  db_instance_storage = 100
  vswitch_id          = alicloud_vswitch.db.id
  ssl_action          = "Close"
}
