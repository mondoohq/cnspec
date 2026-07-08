extension microsoftGraphV1

resource scopedMfa 'Microsoft.Graph/conditionalAccessPolicies@v1.0' = {
  displayName: 'Require MFA for finance group'
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
      includeGroups: [
        '11111111-2222-3333-4444-555555555555'
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
