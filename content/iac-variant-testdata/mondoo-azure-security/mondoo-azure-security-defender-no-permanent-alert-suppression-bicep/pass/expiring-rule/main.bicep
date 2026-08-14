resource suppression 'Microsoft.Security/alertsSuppressionRules@2019-01-01-preview' = {
  name: 'suppress-known-scanner'
  properties: {
    alertType: 'SuspiciousAuthenticationActivity'
    reason: 'Known internal vulnerability scanner'
    state: 'Enabled'
    expirationDateUtc: '2026-12-31T23:59:59Z'
    comment: 'Reviewed quarterly by the security team'
  }
}
