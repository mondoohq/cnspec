# Non-compliant: no configuration block, so Fargate ephemeral storage is not CMK-encrypted.
resource "aws_ecs_cluster" "fail_example" {
  name = "fail-example"
}
