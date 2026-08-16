# Non-compliant: session duration is 13 hours (> 12).
resource "aws_ssoadmin_permission_set" "fail_example" {
  name             = "example"
  instance_arn     = "arn:aws:sso:::instance/ssoins-example"
  session_duration = "PT13H"
}
