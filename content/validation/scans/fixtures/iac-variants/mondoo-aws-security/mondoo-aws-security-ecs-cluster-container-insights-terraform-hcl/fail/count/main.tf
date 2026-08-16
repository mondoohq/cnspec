# Non-compliant: a counted cluster with Container Insights disabled.
resource "aws_ecs_cluster" "counted" {
  count = 2
  name  = "counted-${count.index}"

  setting {
    name  = "containerInsights"
    value = "disabled"
  }
}
