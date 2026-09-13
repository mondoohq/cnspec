# TLS is on, but with no vswitch_id the provider creates the instance on the
# classic network instead of inside a VPC.
resource "alicloud_kvstore_instance" "cache" {
  db_instance_name = "prod-redis"
  instance_class   = "redis.master.small.default"
  instance_type    = "Redis"
  engine_version   = "7.0"
  ssl_enable       = "Enable"
}
