resource "databricks_cluster" "analytics" {
  cluster_name  = "analytics"
  spark_version = "15.4.x-scala2.12"
  node_type_id  = "i3.xlarge"
  num_workers   = 2
  spark_conf = {
    "spark.hadoop.fs.s3a.secret.key" = "wJalrXUtnFEMIK7MDENGbPxRfiCYEXAMPLEKEY"
  }
}
