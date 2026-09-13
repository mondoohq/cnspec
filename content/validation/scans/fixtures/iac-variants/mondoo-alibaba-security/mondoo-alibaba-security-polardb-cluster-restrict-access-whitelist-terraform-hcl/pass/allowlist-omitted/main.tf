# With no allowlist declared the cluster keeps the service default of
# 127.0.0.1, which admits no remote client.
resource "alicloud_polardb_cluster" "prod" {
  db_type       = "MySQL"
  db_version    = "8.0"
  db_node_class = "polar.mysql.x4.large"
  pay_type      = "PostPaid"
  vswitch_id    = alicloud_vswitch.db.id
}
