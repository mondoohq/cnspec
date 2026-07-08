# Non-compliant: two environments; exactly one omits enhanced health reporting.
# .all() over resources must still fail.
resource "aws_elastic_beanstalk_environment" "good" {
  name                = "good"
  application         = "my-app"
  solution_stack_name = "64bit Amazon Linux 2 v3.5.0 running Docker"

  setting {
    namespace = "aws:elasticbeanstalk:healthreporting:system"
    name      = "SystemType"
    value     = "enhanced"
  }
}

resource "aws_elastic_beanstalk_environment" "bad" {
  name                = "bad"
  application         = "my-app"
  solution_stack_name = "64bit Amazon Linux 2 v3.5.0 running Docker"

  setting {
    namespace = "aws:elasticbeanstalk:healthreporting:system"
    name      = "SystemType"
    value     = "basic"
  }
}
