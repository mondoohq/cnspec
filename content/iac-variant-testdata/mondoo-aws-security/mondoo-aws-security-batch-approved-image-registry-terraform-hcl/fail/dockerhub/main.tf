resource "aws_batch_job_definition" "app" {
  name = "app"
  type = "container"
  container_properties = jsonencode({
    image  = "docker.io/library/ubuntu:22.04"
    vcpus  = 1
    memory = 512
  })
}
