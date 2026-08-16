resource "aws_batch_job_definition" "app" {
  name = "app"
  type = "multinode"
  node_properties = jsonencode({
    mainNode = 0
    numNodes = 2
    nodeRangeProperties = [
      {
        targetNodes = "0:"
        container = {
          image  = "docker.io/library/busybox:latest"
          vcpus  = 1
          memory = 512
        }
      }
    ]
  })
}
