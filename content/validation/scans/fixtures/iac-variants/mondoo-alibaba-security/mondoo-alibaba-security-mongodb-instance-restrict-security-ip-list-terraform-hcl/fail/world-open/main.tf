# Internet-wide access to a document store, which is how unsecured MongoDB
# deployments end up in breach reports.
resource "alicloud_mongodb_instance" "prod" {
  engine_version      = "6.0"
  db_instance_class   = "dds.mongo.mid"
  db_instance_storage = 100
  vswitch_id          = alicloud_vswitch.db.id
  security_ip_list    = ["0.0.0.0/0"]
  name                = "prod-mongo"
}
