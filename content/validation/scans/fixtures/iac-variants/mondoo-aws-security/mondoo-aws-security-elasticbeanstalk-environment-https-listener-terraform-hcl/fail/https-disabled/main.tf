# Non-compliant: HTTPS listener on port 443 disabled.
resource "aws_elastic_beanstalk_environment" "fail_example" {
  name                = "fail-example"
  application         = "my-app"
  solution_stack_name = "64bit Amazon Linux 2 v3.5.0 running Docker"

  setting {
    namespace = "aws:elbv2:listener:443"
    name      = "ListenerEnabled"
    value     = "false"
  }
}
