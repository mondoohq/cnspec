window.BENCHMARK_DATA = {
  "lastUpdate": 1751977912067,
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
      }
    ]
  }
}