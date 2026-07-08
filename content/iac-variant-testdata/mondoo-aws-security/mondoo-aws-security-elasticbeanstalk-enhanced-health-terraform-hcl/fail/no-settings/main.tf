# Non-compliant: no health-reporting setting configured, so it stays basic.
resource "aws_elastic_beanstalk_environment" "fail_example" {
  name                = "fail-example"
  application         = "my-app"
  solution_stack_name = "64bit Amazon Linux 2 v3.5.0 running Docker"
}
