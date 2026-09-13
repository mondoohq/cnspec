# With security_ip_list omitted the provider creates the instance with an
# allowlist of 127.0.0.1, which admits no remote client.
resource "alicloud_mongodb_instance" "prod" {
  engine_version      = "6.0"
  db_instance_class   = "dds.mongo.mid"
  db_instance_storage = 100
  vswitch_id          = alicloud_vswitch.db.id
  name                = "prod-mongo"
}
