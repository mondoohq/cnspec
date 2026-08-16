# Non-compliant: Container Insights disabled via a dynamic setting block.
variable "cluster_settings" {
  type    = list(object({ name = string, value = string }))
  default = [{ name = "containerInsights", value = "disabled" }]
}

resource "aws_ecs_cluster" "fail_dynamic" {
  name = "fail-dynamic"

  dynamic "setting" {
    for_each = var.cluster_settings
    content {
      name  = setting.value.name
      value = setting.value.value
    }
  }
}
