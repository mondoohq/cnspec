# Non-compliant: cluster has no log_uri, so logging is disabled.
resource "aws_emr_cluster" "fail_example" {
  name          = "fail-cluster"
  release_label = "emr-6.10.0"
  service_role  = "EMR_DefaultRole"
}
