extension microsoftGraphV1

resource userRiskPolicy 'Microsoft.Graph/conditionalAccessPolicies@v1.0' = {
  displayName: 'Require MFA for high user risk'
  state: 'disabled'
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
    operator: 'OR'
    builtInControls: [
      'mfa'
    ]
  }
}
