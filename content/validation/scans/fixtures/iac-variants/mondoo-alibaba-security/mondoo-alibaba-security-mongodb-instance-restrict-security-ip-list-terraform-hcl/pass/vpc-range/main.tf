resource "alicloud_mongodb_instance" "prod" {
  engine_version      = "6.0"
  db_instance_class   = "dds.mongo.mid"
  db_instance_storage = 100
  vswitch_id          = alicloud_vswitch.db.id
  security_ip_list    = ["10.0.0.0/16"]
  name                = "prod-mongo"
}
