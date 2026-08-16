# The public endpoint is on, but only the corporate egress range can reach it.
resource "alicloud_elasticsearch_instance" "logs" {
  instance_charge_type = "PostPaid"
  data_node_amount     = 2
  data_node_spec       = "elasticsearch.sn2ne.large"
  data_node_disk_size  = 20
  data_node_disk_type  = "cloud_ssd"
  version              = "7.10_with_X-Pack"
  vswitch_id           = "vsw-example"
  password             = var.elasticsearch_password
  enable_public        = true
  public_whitelist     = ["203.0.113.0/24"]
}
