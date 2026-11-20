// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

targetScope = 'subscription'

resource definition 'Microsoft.ManagedServices/registrationDefinitions@2022-10-01' = {
  name: guid('managed-operations')
  properties: {
    registrationDefinitionName: 'Managed operations'
    managedByTenantId: '00000000-0000-0000-0000-000000000001'
    authorizations: [
      {
        principalId: '00000000-0000-0000-0000-000000000002'
        principalIdDisplayName: 'Provider operators'
        roleDefinitionId: '43d0d8ad-25c7-4714-9337-8ba259a9fe05'
      }
      {
        principalId: '00000000-0000-0000-0000-000000000003'
        principalIdDisplayName: 'Provider automation'
        roleDefinitionId: '18d7d88d-d35e-4fb5-a5c3-7773c20a72d9'
        delegatedRoleDefinitionIds: [
          'b24988ac-6180-42a0-ab88-20f7382dd24c'
        ]
      }
    ]
  }
}
