# Non-compliant: a single-segment path like "library/ubuntu" has no registry host (the
# first component is not a hostname), so it resolves to docker.io — same as native.
resource "aws_batch_job_definition" "app" {
  name = "app"
  type = "container"
  container_properties = jsonencode({
    image  = "library/ubuntu:22.04"
    vcpus  = 1
    memory = 512
  })
}
