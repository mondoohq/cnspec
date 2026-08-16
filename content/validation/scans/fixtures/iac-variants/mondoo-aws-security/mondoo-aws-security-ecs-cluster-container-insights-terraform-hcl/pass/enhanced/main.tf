# Compliant: Container Insights enabled in enhanced-observability mode.
resource "aws_ecs_cluster" "pass_example" {
  name = "pass-example"

  setting {
    name  = "containerInsights"
    value = "enhanced"
  }
}
