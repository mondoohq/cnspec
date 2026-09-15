# SBOM testing

Pins the package list `cnspec sbom` reports for one image per packaging style.
Each image is covered by a **recording** — a captured scan — and a **golden**, the
CLI output replaying that recording produces.

The recording supplies the package data, so a failure here means cnspec changed
how it reads packages — not that an image was updated upstream. It does **not**
remove the Docker dependency: cnspec still resolves the image to identify the
asset, and replaying an image that is not present locally reports nothing. The
suite therefore keeps the `debugtest` tag and stays out of `go test ./...`.

## Running it

The suite carries the `debugtest` tag and stays out of `go test ./...`:

```bash
go test -tags debugtest -count=1 ./test/sbom/
```

## Why recordings

It used to scan the images live, which made it wrong in three ways at once and
said nothing about any of them:

- the goldens were recorded on x86_64, so every apk and dpkg line mismatched on
  any other architecture and the suite could not pass there at all;
- the tags are mutable, so an upstream security update broke goldens that were
  still correct about cnspec — the package data came from whatever the tag
  pointed at that day;
- and a real parsing regression looked exactly like either of those.

Recording the package data removes all three. The image is still resolved
through Docker, so this fixes what is compared, not what the suite depends on. The trade is that a recording pins the
architecture of the image it captured, so the goldens describe that architecture
rather than the host's. Most were captured on arm64; `ubi7/ubi-minimal` publishes
no arm64 manifest and was pulled with `--platform linux/amd64`, and
`amazonlinux:2023` was captured from an amd64 image. That is a property of each
fixture, not of the machine running the test — which is the point.

## Refreshing a recording

Do this when cnspec legitimately changes what it reports — a provider fix, a new
field — not to make a red suite go green. Read the golden diff first; it is the
change under review.

```bash
go build -o /tmp/cnspec ./apps/cnspec
cd test/sbom

# 1. recapture (needs the image and a working Docker daemon)
/tmp/cnspec sbom docker <image> --record testdata/<name>-recording.json

# 2. regenerate the golden from that recording
/tmp/cnspec sbom docker <image> --use-recording testdata/<name>-recording.json \
  > testdata/<name>-cli.txt
```

`<name>` is the image reference with `:`, `.` and `/` replaced by `-`, e.g.
`ubuntu:22.04` → `ubuntu-22-04`.

## Adding an image

Run the two commands above, then add the image to `recordedImages` in
`sbom_test.go`. `TestSbomCorpusIsComplete` fails if the list and `testdata/`
disagree, so a recording cannot be added or dropped without the corpus being
updated to match — a suite that can quietly shrink reports success for the checks
it stopped running.

## Coverage

All 40 images are recorded, ~5.7MB of testdata:

| Packaging | Images |
|---|---|
| apk    | alpine 3.16, 3.17, 3.18, 3.19 |
| dpkg   | debian 7, 8, 9, 10, 11, 12 · ubuntu 14.04, 16.04, 18.04, 20.04, 22.04 |
| rpm    | almalinux 8.9, 9.3 · amazonlinux 2, 2023 · centos 7, 8 · fedora 37, 38, 39, 40 · openSUSE Leap 15.5, 42.3 · openSUSE Tumbleweed · Oracle Linux 8.9, 9 · RHEL UBI 7, 8 · Rocky Linux 8.9, 9.3 · SLES 12, 15 |
| tdnf   | photon 3.0, 4.0, 5.0 |

Recapturing needs a Docker Hub login: unauthenticated pulls hit the rate limit
well before 40 images.
