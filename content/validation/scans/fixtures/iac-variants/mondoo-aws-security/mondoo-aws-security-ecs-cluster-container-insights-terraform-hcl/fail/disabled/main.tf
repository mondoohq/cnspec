# Non-compliant: Container Insights explicitly disabled.
resource "aws_ecs_cluster" "fail_example" {
  name = "fail-example"

  setting {
    name  = "containerInsights"
    value = "disabled"
  }
}
