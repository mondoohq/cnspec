# TDE cannot be turned on after the cluster is created, so shipping it disabled
# means a migration later.
resource "alicloud_polardb_cluster" "prod" {
  db_type       = "MySQL"
  db_version    = "8.0"
  db_node_class = "polar.mysql.x4.large"
  pay_type      = "PostPaid"
  vswitch_id    = alicloud_vswitch.db.id
  tde_status    = "Disabled"
}
