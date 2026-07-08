# Compliant: DocumentDB cluster exports audit logs to CloudWatch.
resource "aws_docdb_cluster" "pass_example" {
  cluster_identifier              = "pass-example"
  master_username                 = "admin"
  master_password                 = "mustbeeightchars"
  enabled_cloudwatch_logs_exports = ["audit", "profiler"]
}
