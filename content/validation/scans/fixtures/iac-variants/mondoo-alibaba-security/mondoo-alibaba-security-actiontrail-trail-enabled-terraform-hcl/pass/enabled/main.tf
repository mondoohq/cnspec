resource "alicloud_actiontrail_trail" "org_audit" {
  trail_name      = "org-audit"
  oss_bucket_name = alicloud_oss_bucket.audit.bucket
  oss_key_prefix  = "actiontrail/"
  event_rw        = "All"
  trail_region    = "All"
  status          = "Enable"
}
