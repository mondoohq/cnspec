window.BENCHMARK_DATA = {
  "lastUpdate": 1752011315333,
  "repoUrl": "https://github.com/mondoohq/cnspec",
  "entries": {
    "Benchmark": [
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "1ba49189edf085afd3a09b06d472d9f2211e84df",
          "message": "Correct benchmark workflow to compare PRs against main",
          "timestamp": "2025-07-08T09:12:39Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1725/commits/1ba49189edf085afd3a09b06d472d9f2211e84df"
        },
        "date": 1751977119795,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20915,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "60811 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20915,
            "unit": "ns/op",
            "extra": "60811 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "60811 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60811 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19451,
            "unit": "ns/op\t    4900 B/op\t      71 allocs/op",
            "extra": "57562 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19451,
            "unit": "ns/op",
            "extra": "57562 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4900,
            "unit": "B/op",
            "extra": "57562 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "57562 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "igor@mondoo.com",
            "name": "Igor Komlew",
            "username": "glower"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "fde18ab8fc51218eee5ea123aa038c4a390bfe4b",
          "message": "Correct benchmark workflow to compare PRs against main (#1725)",
          "timestamp": "2025-07-08T14:29:05+02:00",
          "tree_id": "8becddf618503f80a806682de5840f4be02860dd",
          "url": "https://github.com/mondoohq/cnspec/commit/fde18ab8fc51218eee5ea123aa038c4a390bfe4b"
        },
        "date": 1751977911446,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19692,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "63902 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19692,
            "unit": "ns/op",
            "extra": "63902 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "63902 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "63902 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19406,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "58406 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19406,
            "unit": "ns/op",
            "extra": "58406 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "58406 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58406 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "96bcefe307c6f2acedbbf26838a5ad84bd5452a1",
          "message": "Enable runtime auto-update configuration for LocalScanner",
          "timestamp": "2025-07-08T12:29:10Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1685/commits/96bcefe307c6f2acedbbf26838a5ad84bd5452a1"
        },
        "date": 1751978017439,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 5844499835,
            "unit": "ns/op\t681353032 B/op\t15544441 allocs/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 5844499835,
            "unit": "ns/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 681353032,
            "unit": "B/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 15544441,
            "unit": "allocs/op",
            "extra": "1 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 34508995,
            "unit": "ns/op\t15555538 B/op\t  108431 allocs/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 34508995,
            "unit": "ns/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 15555538,
            "unit": "B/op",
            "extra": "32 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 108431,
            "unit": "allocs/op",
            "extra": "32 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "salim@afiunemaya.com.mx",
            "name": "Salim Afiune Maya",
            "username": "afiune"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "7393da96a8ab3b9b28f3a6f0cc6c2c40d5be2048",
          "message": "🐛 onboarding: ms365 does not depend on a subscription (#1722)",
          "timestamp": "2025-07-08T06:37:11-07:00",
          "tree_id": "4a90cc9550386d78afc20c8d74e8a37b94d13dce",
          "url": "https://github.com/mondoohq/cnspec/commit/7393da96a8ab3b9b28f3a6f0cc6c2c40d5be2048"
        },
        "date": 1751981864068,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 18821,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "60248 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 18821,
            "unit": "ns/op",
            "extra": "60248 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "60248 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "60248 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20228,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "54308 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20228,
            "unit": "ns/op",
            "extra": "54308 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "54308 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "54308 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "2e0475111dd2fe53df12dd880066bc1084ad74de",
          "message": "🐛 onboarding: ms365 requires `Policy.Read.All` in MsGraph",
          "timestamp": "2025-07-08T13:37:16Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1723/commits/2e0475111dd2fe53df12dd880066bc1084ad74de"
        },
        "date": 1751983664141,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20719,
            "unit": "ns/op\t    4901 B/op\t      71 allocs/op",
            "extra": "67881 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20719,
            "unit": "ns/op",
            "extra": "67881 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4901,
            "unit": "B/op",
            "extra": "67881 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "67881 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 21102,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "66261 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 21102,
            "unit": "ns/op",
            "extra": "66261 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "66261 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "66261 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "salim@afiunemaya.com.mx",
            "name": "Salim Afiune Maya",
            "username": "afiune"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "2d0db013b7a56c2ff3dd331b6af38ee7e065c6ac",
          "message": "🐛 onboarding: ms365 requires `Policy.Read.All` in MsGraph (#1723)\n\nA few queries that were failing:\n\n```\nquery=h2/skzZUdl0=\nmicrosoft.conditionalAccess.policies.where(conditions.users.includeRoles.length > 0) {\n  grantControls.authenticationStrength.displayName == 'Phishing-resistant MFA'\n}\n\nquery=deTFR6B+GS4=\nmicrosoft.conditionalAccess.policies.where(conditions.applications.includeApplications.contains('Microsoft Intune Enrollment')) {\n  sessionControls.signInFrequency.frequencyInterval == 'everyTime' &&\n  sessionControls.signInFrequency.isEnabled == true &&\n  state == 'enabled'\n}\n\nquery=RFe235yOM3g=\nmicrosoft.conditionalAccess.policies.where(conditions.signInRiskLevels == 'high' || conditions.signInRiskLevels == 'medium') {\n  grantControls.builtInControls.contains('mfa')\n  sessionControls.signInFrequency.frequencyInterval == 'everyTime'\n  state == 'enabled'\n}\n\nquery=keqGtCd5aLY=\nmicrosoft.conditionalAccess.policies.where(conditions.userRiskLevels.contains('high')) {\n  grantControls.builtInControls.contains('mfa')\n  grantControls.builtInControls.contains('passwordChange')\n  sessionControls.signInFrequency.frequencyInterval == 'everyTime'\n  state == 'enabled'\n}\n```\n\n---------\n\nSigned-off-by: Salim Afiune Maya <afiune@mondoo.com>",
          "timestamp": "2025-07-08T07:14:29-07:00",
          "tree_id": "a950eb97087851a8c864ff7a6f5f1ee6cfe4ebaf",
          "url": "https://github.com/mondoohq/cnspec/commit/2d0db013b7a56c2ff3dd331b6af38ee7e065c6ac"
        },
        "date": 1751984105720,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20423,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "62812 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20423,
            "unit": "ns/op",
            "extra": "62812 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "62812 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "62812 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19709,
            "unit": "ns/op\t    4899 B/op\t      71 allocs/op",
            "extra": "54590 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19709,
            "unit": "ns/op",
            "extra": "54590 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4899,
            "unit": "B/op",
            "extra": "54590 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "54590 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "email": "41898282+github-actions[bot]@users.noreply.github.com",
            "name": "github-actions[bot]",
            "username": "github-actions[bot]"
          },
          "committer": {
            "email": "noreply@github.com",
            "name": "GitHub",
            "username": "web-flow"
          },
          "distinct": true,
          "id": "3fe1b7509f5315ecc63804afc43b49aba9b9ac8d",
          "message": "🧹 Bump cnquery to v11.62.1 (#1727)\n\nCo-authored-by: Mondoo Tools <tools@mondoo.com>",
          "timestamp": "2025-07-08T15:52:34Z",
          "tree_id": "971a6b8f099dbf911c06b83881a5edcfacfb50f6",
          "url": "https://github.com/mondoohq/cnspec/commit/3fe1b7509f5315ecc63804afc43b49aba9b9ac8d"
        },
        "date": 1751990131344,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20973,
            "unit": "ns/op\t    4893 B/op\t      71 allocs/op",
            "extra": "59000 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20973,
            "unit": "ns/op",
            "extra": "59000 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4893,
            "unit": "B/op",
            "extra": "59000 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "59000 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20683,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "59833 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20683,
            "unit": "ns/op",
            "extra": "59833 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "59833 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "59833 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "91ccb4d2d9fb9303f20b85a0fb78cf5396b5a9d0",
          "message": "⭐ group reporting in CLI",
          "timestamp": "2025-07-08T15:52:39Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1726/commits/91ccb4d2d9fb9303f20b85a0fb78cf5396b5a9d0"
        },
        "date": 1752008165961,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 18831,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "56079 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 18831,
            "unit": "ns/op",
            "extra": "56079 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "56079 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "56079 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19802,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "53580 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19802,
            "unit": "ns/op",
            "extra": "53580 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "53580 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "53580 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "b37441b988405af441722d4078accab108fc7f83",
          "message": "⭐ group reporting in CLI",
          "timestamp": "2025-07-08T15:52:39Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1726/commits/b37441b988405af441722d4078accab108fc7f83"
        },
        "date": 1752008414610,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20118,
            "unit": "ns/op\t    4892 B/op\t      71 allocs/op",
            "extra": "58971 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20118,
            "unit": "ns/op",
            "extra": "58971 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4892,
            "unit": "B/op",
            "extra": "58971 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58971 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20008,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "63955 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20008,
            "unit": "ns/op",
            "extra": "63955 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "63955 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "63955 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "199481b4cb5ce7b736a0877c9633ccada8dc3917",
          "message": "⭐ group reporting in CLI",
          "timestamp": "2025-07-08T15:52:39Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1726/commits/199481b4cb5ce7b736a0877c9633ccada8dc3917"
        },
        "date": 1752008884477,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20065,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "62870 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20065,
            "unit": "ns/op",
            "extra": "62870 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "62870 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "62870 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 19534,
            "unit": "ns/op\t    4894 B/op\t      71 allocs/op",
            "extra": "58615 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 19534,
            "unit": "ns/op",
            "extra": "58615 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4894,
            "unit": "B/op",
            "extra": "58615 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "58615 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "9ea709735c3e24c14de0bd5d6e06dc9e9a0e6a71",
          "message": "⚙️  onboarding: refactor ms365 permissions",
          "timestamp": "2025-07-08T15:52:39Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1728/commits/9ea709735c3e24c14de0bd5d6e06dc9e9a0e6a71"
        },
        "date": 1752009418943,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 20587,
            "unit": "ns/op\t    4897 B/op\t      71 allocs/op",
            "extra": "64575 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 20587,
            "unit": "ns/op",
            "extra": "64575 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4897,
            "unit": "B/op",
            "extra": "64575 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "64575 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 20881,
            "unit": "ns/op\t    4890 B/op\t      71 allocs/op",
            "extra": "64989 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 20881,
            "unit": "ns/op",
            "extra": "64989 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4890,
            "unit": "B/op",
            "extra": "64989 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "64989 times\n4 procs"
          }
        ]
      },
      {
        "commit": {
          "author": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "committer": {
            "name": "mondoohq",
            "username": "mondoohq"
          },
          "id": "cd92de53c3428c3938485e2091a98c3aee2f794f",
          "message": "⚙️  onboarding: refactor ms365 permissions",
          "timestamp": "2025-07-08T15:52:39Z",
          "url": "https://github.com/mondoohq/cnspec/pull/1728/commits/cd92de53c3428c3938485e2091a98c3aee2f794f"
        },
        "date": 1752011314930,
        "tool": "go",
        "benches": [
          {
            "name": "BenchmarkScan_SingleAsset",
            "value": 19741,
            "unit": "ns/op\t    4898 B/op\t      71 allocs/op",
            "extra": "65762 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - ns/op",
            "value": 19741,
            "unit": "ns/op",
            "extra": "65762 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - B/op",
            "value": 4898,
            "unit": "B/op",
            "extra": "65762 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_SingleAsset - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "65762 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets",
            "value": 22178,
            "unit": "ns/op\t    4895 B/op\t      71 allocs/op",
            "extra": "47852 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - ns/op",
            "value": 22178,
            "unit": "ns/op",
            "extra": "47852 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - B/op",
            "value": 4895,
            "unit": "B/op",
            "extra": "47852 times\n4 procs"
          },
          {
            "name": "BenchmarkScan_MultipleAssets - allocs/op",
            "value": 71,
            "unit": "allocs/op",
            "extra": "47852 times\n4 procs"
          }
        ]
      }
    ]
  }
}