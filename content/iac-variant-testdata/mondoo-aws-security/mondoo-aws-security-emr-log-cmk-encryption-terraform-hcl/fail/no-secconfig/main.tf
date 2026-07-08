# Non-compliant: cluster logs to S3 but has no security configuration for log encryption.
resource "aws_emr_cluster" "fail_example" {
  name          = "fail-cluster"
  release_label = "emr-6.10.0"
  service_role  = "EMR_DefaultRole"
  log_uri       = "s3://example-logs/emr/"
}
