# Non-compliant: the ternary's active (default) branch disables Container
# Insights. A real config defaulting insights off looks exactly like this.
variable "enable_insights" {
  type    = bool
  default = false
}

resource "aws_ecs_cluster" "ternary" {
  name = "ternary"
  setting {
    name  = "containerInsights"
    value = var.enable_insights ? "enabled" : "disabled"
  }
}
