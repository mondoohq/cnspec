# Non-compliant: environment settings present, but no HTTPS listener on port 443.
resource "aws_elastic_beanstalk_environment" "fail_example" {
  name                = "fail-example"
  application         = "my-app"
  solution_stack_name = "64bit Amazon Linux 2 v3.5.0 running Docker"

  setting {
    namespace = "aws:elasticbeanstalk:environment"
    name      = "LoadBalancerType"
    value     = "application"
  }
}
