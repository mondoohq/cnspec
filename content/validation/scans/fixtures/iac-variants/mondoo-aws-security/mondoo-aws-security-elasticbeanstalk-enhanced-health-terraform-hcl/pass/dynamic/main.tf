# Compliant: enhanced health reporting enabled, but settings are generated via a
# dynamic block. Elastic Beanstalk environments very commonly build their
# setting blocks dynamically from a variable.
variable "settings" {
  type = list(object({ namespace = string, name = string, value = string }))
  default = [
    {
      namespace = "aws:elasticbeanstalk:healthreporting:system"
      name      = "SystemType"
      value     = "enhanced"
    },
  ]
}

resource "aws_elastic_beanstalk_environment" "pass_example" {
  name                = "pass-example"
  application         = "my-app"
  solution_stack_name = "64bit Amazon Linux 2 v3.5.0 running Docker"

  dynamic "setting" {
    for_each = var.settings
    content {
      namespace = setting.value.namespace
      name      = setting.value.name
      value     = setting.value.value
    }
  }
}
