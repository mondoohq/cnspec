extension microsoftGraphV1

resource allUsersMfa 'Microsoft.Graph/conditionalAccessPolicies@v1.0' = {
  displayName: 'Require MFA for all users'
  state: 'disabled'
  conditions: {
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
