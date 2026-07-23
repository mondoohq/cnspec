resource "aws_neptune_cluster" "example" {
  cluster_identifier     = "example"
  vpc_security_group_ids = ["sg-0123456789abcdef0"]
}
