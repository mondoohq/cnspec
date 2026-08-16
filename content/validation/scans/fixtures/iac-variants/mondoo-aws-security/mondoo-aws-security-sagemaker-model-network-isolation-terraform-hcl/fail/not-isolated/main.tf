# Non-compliant: model does not enable network isolation.
resource "aws_sagemaker_model" "fail_example" {
  name                     = "example-model"
  execution_role_arn       = "arn:aws:iam::123456789012:role/example"
  enable_network_isolation = false

  primary_container {
    image = "123456789012.dkr.ecr.us-east-1.amazonaws.com/example:latest"
  }
}
