# Compliant: a real third-party registry host that is not one of the blocked public
# registries (docker.io / registry.hub.docker.com / public.ecr.aws).
resource "aws_batch_job_definition" "app" {
  name = "app"
  type = "container"
  container_properties = jsonencode({
    image  = "gcr.io/my-project/app:1.0"
    vcpus  = 1
    memory = 512
  })
}
