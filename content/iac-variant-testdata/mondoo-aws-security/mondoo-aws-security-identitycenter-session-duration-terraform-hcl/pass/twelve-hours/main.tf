# Compliant: session duration is 8 hours (<= 12).
resource "aws_ssoadmin_permission_set" "pass_example" {
  name             = "example"
  instance_arn     = "arn:aws:sso:::instance/ssoins-example"
  session_duration = "PT8H"
}
