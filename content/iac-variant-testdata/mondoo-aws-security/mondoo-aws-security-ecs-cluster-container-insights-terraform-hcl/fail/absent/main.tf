# Non-compliant: no setting block, so Container Insights is not enabled.
resource "aws_ecs_cluster" "fail_example" {
  name = "fail-example"
}
