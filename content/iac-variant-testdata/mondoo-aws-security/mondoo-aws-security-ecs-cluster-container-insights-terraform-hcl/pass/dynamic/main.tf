# Compliant: Container Insights enabled, but the setting is emitted via a
# dynamic block (a common pattern when settings are data-driven).
variable "cluster_settings" {
  type    = list(object({ name = string, value = string }))
  default = [{ name = "containerInsights", value = "enabled" }]
}

resource "aws_ecs_cluster" "pass_dynamic" {
  name = "pass-dynamic"

  dynamic "setting" {
    for_each = var.cluster_settings
    content {
      name  = setting.value.name
      value = setting.value.value
    }
  }
}
