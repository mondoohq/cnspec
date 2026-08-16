extension microsoftGraphV1

resource allUsersMfa 'Microsoft.Graph/conditionalAccessPolicies@v1.0' = {
  displayName: 'Require MFA for all users'
  state: 'enabled'
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
