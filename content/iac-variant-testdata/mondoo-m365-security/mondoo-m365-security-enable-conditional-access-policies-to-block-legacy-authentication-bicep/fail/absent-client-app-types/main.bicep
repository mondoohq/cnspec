extension microsoftGraphV1

resource blockPolicy 'Microsoft.Graph/conditionalAccessPolicies@v1.0' = {
  displayName: 'Block risky sign-ins'
  state: 'enabled'
  conditions: {
    signInRiskLevels: [
      'high'
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
      'block'
    ]
  }
}
