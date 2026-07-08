# Non-compliant: DocumentDB cluster does not export audit logs.
resource "aws_docdb_cluster" "fail_example" {
  cluster_identifier              = "fail-example"
  master_username                 = "admin"
  master_password                 = "mustbeeightchars"
  enabled_cloudwatch_logs_exports = ["profiler"]
}
