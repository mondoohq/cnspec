extension microsoftGraphV1

resource userRiskPolicy 'Microsoft.Graph/conditionalAccessPolicies@v1.0' = {
  displayName: 'Restrict high user risk sessions'
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
    operator: 'OR'
    builtInControls: [
      'passwordChange'
    ]
  }
}
