extension microsoftGraphV1

resource adminPolicy 'Microsoft.Graph/conditionalAccessPolicies@v1.0' = {
  displayName: 'Require compliant device for administrators'
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
      includeRoles: [
        '62e90394-69f5-4237-9190-012177145e10'
      ]
    }
  }
  grantControls: {
    operator: 'OR'
    builtInControls: [
      'compliantDevice'
    ]
  }
}
