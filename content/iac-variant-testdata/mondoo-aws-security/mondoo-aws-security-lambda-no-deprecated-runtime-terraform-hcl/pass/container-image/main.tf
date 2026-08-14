# Image-based functions declare no runtime at all, which the check treats as
# out of scope rather than as a deprecated runtime.
resource "aws_lambda_function" "processor" {
  function_name = "event-processor"
  role          = aws_iam_role.lambda.arn
  package_type  = "Image"
  image_uri     = "123456789012.dkr.ecr.us-east-1.amazonaws.com/processor:1.4.2"
}
