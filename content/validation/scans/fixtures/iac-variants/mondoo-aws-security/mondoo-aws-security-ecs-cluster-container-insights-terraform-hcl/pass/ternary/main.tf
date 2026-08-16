# Compliant intent: Container Insights value chosen via a ternary; the active
# (default) branch is "enabled".
variable "enable_insights" {
  type    = bool
  default = true
}

resource "aws_ecs_cluster" "ternary" {
  name = "ternary"
  setting {
    name  = "containerInsights"
    value = var.enable_insights ? "enabled" : "disabled"
  }
}
