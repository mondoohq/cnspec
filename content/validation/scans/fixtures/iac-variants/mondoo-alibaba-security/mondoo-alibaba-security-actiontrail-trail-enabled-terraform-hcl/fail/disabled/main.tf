# A disabled trail exists in the console but records nothing, which is worse
# than no trail because it looks like coverage.
resource "alicloud_actiontrail_trail" "org_audit" {
  trail_name      = "org-audit"
  oss_bucket_name = alicloud_oss_bucket.audit.bucket
  event_rw        = "All"
  trail_region    = "All"
  status          = "Disable"
}
