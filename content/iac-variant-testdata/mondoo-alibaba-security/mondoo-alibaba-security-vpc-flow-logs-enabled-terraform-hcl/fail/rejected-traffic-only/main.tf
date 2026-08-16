# The flow log is active but records only rejected traffic, so the connections
# that actually succeeded leave no record.
resource "alicloud_vpc_flow_log" "prod" {
  flow_log_name  = "prod-vpc-flow"
  resource_id    = alicloud_vpc.prod.id
  resource_type  = "VPC"
  traffic_type   = "Drop"
  project_name   = "vpc-flow-logs"
  log_store_name = "prod-vpc"
  status         = "Active"
}
