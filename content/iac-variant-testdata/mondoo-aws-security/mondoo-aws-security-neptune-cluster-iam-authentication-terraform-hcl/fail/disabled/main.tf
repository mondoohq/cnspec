# Non-compliant: IAM database authentication is disabled.
resource "aws_neptune_cluster" "fail_example" {
  cluster_identifier                  = "example"
  iam_database_authentication_enabled = false
}
