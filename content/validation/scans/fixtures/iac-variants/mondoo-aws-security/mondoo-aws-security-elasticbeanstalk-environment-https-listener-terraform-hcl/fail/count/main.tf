# Non-compliant: a counted environment disables the 443 listener.
resource "aws_elastic_beanstalk_environment" "violating" {
  count               = 2
  name                = "violating-env-${count.index}"
  application         = "my-app"
  solution_stack_name = "64bit Amazon Linux 2 v3.5.0 running Docker"

  setting {
    namespace = "aws:elbv2:listener:443"
    name      = "ListenerEnabled"
    value     = "false"
  }
}
