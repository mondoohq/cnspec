# Two environments; the second one disables the 443 listener, so .all() must fail.
resource "aws_elastic_beanstalk_environment" "compliant" {
  name                = "compliant-env"
  application         = "my-app"
  solution_stack_name = "64bit Amazon Linux 2 v3.5.0 running Docker"

  setting {
    namespace = "aws:elbv2:listener:443"
    name      = "ListenerEnabled"
    value     = "true"
  }
}

resource "aws_elastic_beanstalk_environment" "violating" {
  name                = "violating-env"
  application         = "my-app"
  solution_stack_name = "64bit Amazon Linux 2 v3.5.0 running Docker"

  setting {
    namespace = "aws:elbv2:listener:443"
    name      = "ListenerEnabled"
    value     = "false"
  }
}
