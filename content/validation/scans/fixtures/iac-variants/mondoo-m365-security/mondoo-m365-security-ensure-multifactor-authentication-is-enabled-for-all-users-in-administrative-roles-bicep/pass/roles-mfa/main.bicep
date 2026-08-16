extension microsoftGraphV1

resource adminMfaPolicy 'Microsoft.Graph/conditionalAccessPolicies@v1.0' = {
  displayName: 'Require MFA for administrators'
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
        '194ae4cb-b126-40b2-bd5b-6091b380977d'
        'f28a1f50-f6e7-4571-818b-6a12f2af6b6c'
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
