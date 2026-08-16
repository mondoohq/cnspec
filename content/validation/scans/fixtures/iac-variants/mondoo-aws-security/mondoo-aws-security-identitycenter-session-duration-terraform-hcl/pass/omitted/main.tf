# Compliant: session_duration is omitted, so the check treats it as acceptable.
resource "aws_ssoadmin_permission_set" "pass_example" {
  name         = "read-only"
  instance_arn = "arn:aws:sso:::instance/ssoins-example"
}
