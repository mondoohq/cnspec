# Compliant: HTTPS listener on port 443 enabled via a dynamic "setting" block.
variable "listener_settings" {
  type = list(object({ namespace = string, name = string, value = string }))
  default = [
    { namespace = "aws:elbv2:listener:443", name = "ListenerEnabled", value = "true" },
  ]
}

resource "aws_elastic_beanstalk_environment" "pass_dynamic" {
  name                = "pass-dynamic"
  application         = "my-app"
  solution_stack_name = "64bit Amazon Linux 2 v3.5.0 running Docker"

  dynamic "setting" {
    for_each = var.listener_settings
    content {
      namespace = setting.value.namespace
      name      = setting.value.name
      value     = setting.value.value
    }
  }
}
