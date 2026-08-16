resource "aws_neptune_cluster" "example" {
  cluster_identifier                  = "example"
  engine                              = "neptune"
  backup_retention_period             = 5
  iam_database_authentication_enabled = true
}
