# Compliant: Container Insights enabled on the cluster.
resource "aws_ecs_cluster" "pass_example" {
  name = "pass-example"

  setting {
    name  = "containerInsights"
    value = "enabled"
  }
}
