# A bare 0.0.0.0 entry is treated as open, the same as 0.0.0.0/0, by the live
# check against the instance.
resource "alicloud_mongodb_instance" "prod" {
  engine_version      = "6.0"
  db_instance_class   = "dds.mongo.mid"
  db_instance_storage = 100
  vswitch_id          = alicloud_vswitch.db.id
  security_ip_list    = ["10.0.0.0/16", "0.0.0.0"]
  name                = "prod-mongo"
}
