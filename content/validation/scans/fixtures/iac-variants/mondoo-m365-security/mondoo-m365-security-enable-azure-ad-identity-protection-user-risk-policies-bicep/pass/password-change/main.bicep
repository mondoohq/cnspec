extension microsoftGraphV1

resource userRiskPolicy 'Microsoft.Graph/conditionalAccessPolicies@v1.0' = {
  displayName: 'Require password change for high user risk'
  state: 'enabled'
  conditions: {
    userRiskLevels: [
      'high'
    ]
    clientAppTypes: [
      'all'
    ]
    applications: {
      includeApplications: [
        'All'
      ]
    }
    users: {
      includeUsers: [
        'All'
      ]
    }
  }
  grantControls: {
    operator: 'AND'
    builtInControls: [
      'mfa'
      'passwordChange'
    ]
  }
}
