resource "aws_neptune_cluster" "example" {
  cluster_identifier     = "example"
  engine                 = "neptune"
  vpc_security_group_ids = []
}
