# Compliant: model has a vpc_config block with subnets.
resource "aws_sagemaker_model" "pass_example" {
  name               = "example-model"
  execution_role_arn = "arn:aws:iam::123456789012:role/example"

  primary_container {
    image = "123456789012.dkr.ecr.us-east-1.amazonaws.com/example:latest"
  }

  vpc_config {
    security_group_ids = ["sg-0123456789abcdef0"]
    subnets            = ["subnet-0123456789abcdef0", "subnet-0123456789abcdef1"]
  }
}
