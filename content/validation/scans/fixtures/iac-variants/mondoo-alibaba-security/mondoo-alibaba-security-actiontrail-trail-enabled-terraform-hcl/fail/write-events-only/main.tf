# The trail is enabled, but it records only write operations. Read operations,
# which is where reconnaissance and data access show up, go unrecorded.
resource "alicloud_actiontrail_trail" "write_only" {
  trail_name      = "write-only-audit"
  oss_bucket_name = alicloud_oss_bucket.audit.bucket
  oss_key_prefix  = "actiontrail/"
  event_rw        = "Write"
  trail_region    = "All"
  status          = "Enable"
}
