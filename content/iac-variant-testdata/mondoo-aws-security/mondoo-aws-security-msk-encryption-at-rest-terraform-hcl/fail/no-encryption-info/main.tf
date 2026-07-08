# Non-compliant: MSK cluster declares no encryption_info block, so data at rest
# is not encrypted with a customer-managed KMS key.
resource "aws_msk_cluster" "fail_example" {
  cluster_name           = "fail-example"
  kafka_version          = "3.5.1"
  number_of_broker_nodes = 3

  broker_node_group_info {
    instance_type   = "kafka.m5.large"
    client_subnets  = ["subnet-aaaa", "subnet-bbbb", "subnet-cccc"]
    security_groups = ["sg-1234"]

    storage_info {
      ebs_storage_info {
        volume_size = 100
      }
    }
  }
}
