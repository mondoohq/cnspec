resource roleAssignment 'Microsoft.Authorization/roleAssignmentScheduleRequests@2022-04-01-preview' = {
  name: '00000000-0000-0000-0000-000000000001'
  properties: {
    principalId: '11111111-1111-1111-1111-111111111111'
    roleDefinitionId: '/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/roleDefinitions/b24988ac-6180-42a0-ab88-20f7382dd24c'
    requestType: 'AdminAssign'
    justification: 'Scheduled maintenance window'
    scheduleInfo: {
      startDateTime: '2026-01-01T00:00:00Z'
      expiration: {
        type: 'AfterDuration'
        duration: 'PT8H'
      }
    }
  }
}
