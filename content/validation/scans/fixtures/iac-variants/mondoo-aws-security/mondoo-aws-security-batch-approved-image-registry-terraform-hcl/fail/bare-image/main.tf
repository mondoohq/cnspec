# Non-compliant: a bare image name has no registry host, so it implicitly pulls from
# Docker Hub (docker.io) — the native check resolves it that way and flags it.
resource "aws_batch_job_definition" "app" {
  name = "app"
  type = "container"
  container_properties = jsonencode({
    image  = "ubuntu:22.04"
    vcpus  = 1
    memory = 512
  })
}
