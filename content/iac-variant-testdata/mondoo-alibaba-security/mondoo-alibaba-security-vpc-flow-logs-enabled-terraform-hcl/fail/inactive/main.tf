# An inactive flow log captures no traffic, leaving no network record for
# incident investigation.
resource "alicloud_vpc_flow_log" "prod" {
  flow_log_name  = "prod-vpc-flow"
  resource_id    = alicloud_vpc.prod.id
  resource_type  = "VPC"
  traffic_type   = "All"
  project_name   = "vpc-flow-logs"
  log_store_name = "prod-vpc"
  status         = "Inactive"
}
