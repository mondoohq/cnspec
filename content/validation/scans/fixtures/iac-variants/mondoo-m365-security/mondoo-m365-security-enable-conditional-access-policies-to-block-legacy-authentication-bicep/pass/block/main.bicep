extension microsoftGraphV1

resource blockLegacyAuth 'Microsoft.Graph/conditionalAccessPolicies@v1.0' = {
  displayName: 'Block legacy authentication'
  state: 'enabled'
  conditions: {
    clientAppTypes: [
      'exchangeActiveSync'
      'other'
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
      'block'
    ]
  }
}
