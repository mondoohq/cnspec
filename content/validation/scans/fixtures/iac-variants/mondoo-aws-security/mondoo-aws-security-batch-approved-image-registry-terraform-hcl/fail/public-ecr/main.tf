resource "aws_batch_job_definition" "app" {
  name = "app"
  type = "container"
  container_properties = jsonencode({
    image  = "public.ecr.aws/amazonlinux/amazonlinux:latest"
    vcpus  = 1
    memory = 512
  })
}
