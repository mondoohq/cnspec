targetScope = 'subscription'

// A disabled rule suppresses nothing, so it needs no expiry. The check scopes
// itself to Enabled rules.
resource suppression 'Microsoft.Security/alertsSuppressionRules@2019-01-01-preview' = {
  name: 'suppress-known-scanner'
  properties: {
    alertType: 'SuspiciousAuthenticationActivity'
    reason: 'Retired scanner, rule kept for reference'
    state: 'Disabled'
  }
}