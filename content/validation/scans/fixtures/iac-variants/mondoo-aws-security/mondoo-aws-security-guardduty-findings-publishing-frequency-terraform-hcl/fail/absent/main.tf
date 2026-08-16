# finding_publishing_frequency omitted; AWS defaults it to SIX_HOURS.
resource "aws_guardduty_detector" "this" {
  enable = true
}
