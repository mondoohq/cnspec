# Non-compliant: iam:PassRole together with ec2:RunInstances lets the principal
# launch an instance carrying a more privileged role. The two actions sit in
# separate statements, which is how this combination usually appears.
resource "aws_iam_policy" "passrole_ec2" {
  name = "passrole-ec2"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      { Effect = "Allow", Action = ["iam:PassRole"], Resource = "*" },
      { Effect = "Allow", Action = ["ec2:RunInstances"], Resource = "*" }
    ]
  })
}
