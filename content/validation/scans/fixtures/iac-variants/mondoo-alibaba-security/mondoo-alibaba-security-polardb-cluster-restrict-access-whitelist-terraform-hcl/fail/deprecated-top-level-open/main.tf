# The deprecated top-level security_ips argument still writes the default
# whitelist group, so an open range here opens the cluster.
resource "alicloud_polardb_cluster" "prod" {
  db_type       = "MySQL"
  db_version    = "8.0"
  db_node_class = "polar.mysql.x4.large"
  pay_type      = "PostPaid"
  vswitch_id    = alicloud_vswitch.db.id
  security_ips  = ["0.0.0.0/0"]
}
