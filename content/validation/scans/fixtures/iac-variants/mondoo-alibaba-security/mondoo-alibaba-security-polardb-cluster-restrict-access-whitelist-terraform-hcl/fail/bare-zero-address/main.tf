# A bare 0.0.0.0 entry in any whitelist group is treated as open, the same as
# 0.0.0.0/0, by the live check against the cluster.
resource "alicloud_polardb_cluster" "prod" {
  db_type       = "MySQL"
  db_version    = "8.0"
  db_node_class = "polar.mysql.x4.large"
  pay_type      = "PostPaid"
  vswitch_id    = alicloud_vswitch.db.id

  db_cluster_ip_array {
    db_cluster_ip_array_name = "default"
    security_ips             = ["10.0.0.0/16"]
  }

  db_cluster_ip_array {
    db_cluster_ip_array_name = "migration"
    security_ips             = ["0.0.0.0"]
  }
}
