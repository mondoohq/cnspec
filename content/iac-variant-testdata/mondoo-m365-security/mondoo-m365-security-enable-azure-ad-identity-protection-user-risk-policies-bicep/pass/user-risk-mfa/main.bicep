extension microsoftGraphV1

resource userRiskPolicy 'Microsoft.Graph/conditionalAccessPolicies@v1.0' = {
  displayName: 'Require MFA for high user risk'
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
      excludeUsers: [
        '00000000-0000-0000-0000-000000000000'
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
