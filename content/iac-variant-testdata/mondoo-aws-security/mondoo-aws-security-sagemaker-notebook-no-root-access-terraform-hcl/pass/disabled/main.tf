# Compliant: root access disabled on the notebook instance.
resource "aws_sagemaker_notebook_instance" "pass_example" {
  name          = "example"
  instance_type = "ml.t3.medium"
  role_arn      = "arn:aws:iam::123456789012:role/example"
  root_access   = "Disabled"
}
