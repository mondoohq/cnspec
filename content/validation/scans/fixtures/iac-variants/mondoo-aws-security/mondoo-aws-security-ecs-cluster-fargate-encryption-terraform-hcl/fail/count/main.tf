# Non-compliant: clusters created with count, none supplying a Fargate KMS key.
resource "aws_ecs_cluster" "fail_example" {
  count = 2
  name  = "fail-example-${count.index}"

  configuration {
    managed_storage_configuration {
    }
  }
}
