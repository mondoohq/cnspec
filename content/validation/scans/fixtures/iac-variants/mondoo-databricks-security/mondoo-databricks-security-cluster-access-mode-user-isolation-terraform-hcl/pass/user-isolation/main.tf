resource "databricks_cluster" "analytics" {
  cluster_name  = "analytics"
  spark_version = "15.4.x-scala2.12"
  node_type_id  = "i3.xlarge"
  num_workers   = 2
  data_security_mode = "USER_ISOLATION"
}
