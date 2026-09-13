# The default group is scoped, but a named group opened during a migration
# admits every address. security_ip_list inside a group is a comma-joined string.
resource "alicloud_mongodb_instance" "prod" {
  engine_version      = "6.0"
  db_instance_class   = "dds.mongo.mid"
  db_instance_storage = 100
  vswitch_id          = alicloud_vswitch.db.id
  security_ip_list    = ["10.0.0.0/16"]
  name                = "prod-mongo"

  security_ip_groups {
    security_ip_group_name = "migration"
    security_ip_list       = "10.1.0.0/16,0.0.0.0/0"
  }
}
