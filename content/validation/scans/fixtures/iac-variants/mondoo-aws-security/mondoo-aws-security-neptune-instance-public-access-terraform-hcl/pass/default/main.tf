resource "aws_neptune_cluster_instance" "example" {
  cluster_identifier = "example"
  instance_class     = "db.r5.large"
  engine             = "neptune"
}
