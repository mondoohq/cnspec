# Two clusters; the second explicitly disables Container Insights.
resource "aws_ecs_cluster" "compliant" {
  name = "compliant"

  setting {
    name  = "containerInsights"
    value = "enabled"
  }
}

resource "aws_ecs_cluster" "violating" {
  name = "violating"

  setting {
    name  = "containerInsights"
    value = "disabled"
  }
}
