targetScope = 'subscription'

// A suppression rule with no expiry silences an alert class forever, and
// nothing forces anyone to revisit whether it is still justified.
resource suppression 'Microsoft.Security/alertsSuppressionRules@2019-01-01-preview' = {
  name: 'suppress-known-scanner'
  properties: {
    alertType: 'SuspiciousAuthenticationActivity'
    reason: 'Known internal vulnerability scanner'
    state: 'Enabled'
  }
}