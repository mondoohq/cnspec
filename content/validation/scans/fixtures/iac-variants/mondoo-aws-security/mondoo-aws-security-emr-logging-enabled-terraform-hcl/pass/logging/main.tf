# Compliant: cluster has a log_uri configured.
resource "aws_emr_cluster" "pass_example" {
  name          = "pass-cluster"
  release_label = "emr-6.10.0"
  service_role  = "EMR_DefaultRole"
  log_uri       = "s3://example-logs/emr/"
}
