# Non-compliant: iam_database_authentication_enabled is omitted, defaulting to false.
resource "aws_neptune_cluster" "fail_example" {
  cluster_identifier = "example"
  engine             = "neptune"
}
