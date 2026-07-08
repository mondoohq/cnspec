resource "aws_batch_job_definition" "app" {
  name = "app"
  type = "container"
  container_properties = jsonencode({
    image      = "123456789012.dkr.ecr.us-east-1.amazonaws.com/app:1.0"
    privileged = false
    vcpus      = 1
    memory     = 512
  })
}
