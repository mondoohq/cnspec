# MongoDB's tde_status uses lowercase values, unlike RDS and Redis; "disabled"
# leaves the collections unencrypted at rest.
resource "alicloud_mongodb_instance" "prod" {
  engine_version      = "6.0"
  db_instance_class   = "dds.mongo.mid"
  db_instance_storage = 100
  vswitch_id          = alicloud_vswitch.db.id
  tde_status          = "disabled"
}
