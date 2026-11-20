// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

targetScope = 'subscription'

resource definition 'Microsoft.ManagedServices/registrationDefinitions@2022-10-01' = {
  name: guid('managed-monitoring')
  properties: {
    registrationDefinitionName: 'Managed monitoring'
    managedByTenantId: '00000000-0000-0000-0000-000000000001'
    authorizations: [
      {
        principalId: '00000000-0000-0000-0000-000000000002'
        principalIdDisplayName: 'Provider operators'
        roleDefinitionId: '43d0d8ad-25c7-4714-9337-8ba259a9fe05'
      }
    ]
  }
}
