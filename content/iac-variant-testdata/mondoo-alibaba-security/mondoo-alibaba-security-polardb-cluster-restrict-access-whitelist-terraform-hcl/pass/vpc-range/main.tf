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
}
