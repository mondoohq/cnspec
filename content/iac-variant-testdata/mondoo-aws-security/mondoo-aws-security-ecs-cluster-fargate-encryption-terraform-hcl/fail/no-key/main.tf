# Non-compliant: managed storage configuration present but no Fargate KMS key.
resource "aws_ecs_cluster" "fail_example" {
  name = "fail-example"

  configuration {
    managed_storage_configuration {
    }
  }
}
