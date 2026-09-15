// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

//go:build debugtest
// +build debugtest

package sbom

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/test"
)

// This suite pins the package list cnspec reports for one image per packaging
// style, against a recorded scan rather than a live one.
//
// It used to scan the images live, which made it wrong in three ways at once and
// said nothing about any of them: the goldens were recorded on x86_64 and could
// not pass anywhere else; the tags are mutable, so every upstream security update
// broke them; and a change in how a package manager is parsed was indistinguishable
// from either. Replaying a recording fixes the input, so a diff here means cnspec
// changed -- which is the only thing this suite can usefully report.
//
// A recording supplies the package data, not the target: cnspec still resolves
// the image through Docker to identify the asset, and replaying an image that is
// not present locally reports nothing. So the suite still needs the images and a
// working Docker daemon, and it keeps the debugtest tag -- it is not hermetic and
// cannot run on every `go test ./...`. What the recording removes is the reason
// the goldens kept breaking: their contents no longer come from whatever the
// mutable tag points at today.
//
// See README.md to refresh a recording or add an image.

var once sync.Once

// setup builds cnspec locally
func setup() {
	if err := exec.Command("go", "build", "../../apps/cnspec/cnspec.go").Run(); err != nil {
		log.Fatalf("building cnspec: %v", err)
	}
}

func TestMain(m *testing.M) {
	ret := m.Run()
	os.Exit(ret)
}

// recordedImages is the corpus: every image with a recording checked in. The
// list is explicit rather than whatever testdata happens to hold, so a recording
// that is deleted, or added without a golden, fails the suite instead of quietly
// shrinking it.
var recordedImages = []string{
	"alpine:3.16",
	"alpine:3.17",
	"alpine:3.18",
	"alpine:3.19",
	"almalinux:8.9",
	"almalinux:9.3",
	"amazonlinux:2",
	"amazonlinux:2023",
	"centos:7",
	"centos:8",
	"debian:7",
	"debian:8",
	"debian:9",
	"debian:10",
	"debian:11",
	"debian:12",
	"fedora:37",
	"fedora:38",
	"fedora:39",
	"fedora:40",
	"opensuse/leap:15.5",
	"opensuse/leap:42.3",
	"opensuse/tumbleweed",
	"oraclelinux:8.9",
	"oraclelinux:9",
	"photon:3.0",
	"photon:4.0",
	"photon:5.0",
	"registry.access.redhat.com/ubi7/ubi-minimal:7.9-1313",
	"registry.access.redhat.com/ubi8/ubi:8.0-122",
	"registry.access.redhat.com/ubi8/ubi:8.9-1107",
	"rockylinux:8.9",
	"rockylinux:9.3",
	"registry.suse.com/bci/bci-base:15.5",
	"registry.suse.com/suse/sles12sp5:6.5.559",
	"ubuntu:14.04",
	"ubuntu:16.04",
	"ubuntu:18.04",
	"ubuntu:20.04",
	"ubuntu:22.04",
}

func TestSbomGeneration(t *testing.T) {
	once.Do(setup)

	for _, img := range recordedImages {
		t.Run(img, func(t *testing.T) {
			testSbomExport(t, img, false)
		})
	}
}

// TestSbomCorpusIsComplete fails when testdata and recordedImages disagree. A
// suite whose corpus can shrink without anyone noticing reports success for
// exactly the checks it stopped running, so the corpus is asserted rather than
// discovered.
func TestSbomCorpusIsComplete(t *testing.T) {
	require.NotEmpty(t, recordedImages, "an empty corpus would pass every check by running none")

	found, err := filepath.Glob("testdata/*-recording.json")
	require.NoError(t, err)

	onDisk := make([]string, 0, len(found))
	for _, path := range found {
		onDisk = append(onDisk, strings.TrimSuffix(filepath.Base(path), "-recording.json"))
	}

	want := make([]string, 0, len(recordedImages))
	for _, img := range recordedImages {
		want = append(want, fixtureName(img))
	}

	sort.Strings(onDisk)
	sort.Strings(want)
	assert.Equal(t, want, onDisk, "recordedImages and testdata/*-recording.json must name the same images")

	for _, img := range recordedImages {
		_, err := os.Stat("testdata/" + fixtureName(img) + "-cli.txt")
		assert.NoError(t, err, "%s has a recording but no golden", img)
	}
}

// fixtureName is the testdata basename for an image reference.
func fixtureName(img string) string {
	name := strings.ReplaceAll(img, ":", "-")
	name = strings.ReplaceAll(name, ".", "-")
	return strings.ReplaceAll(name, "/", "-")
}

func testSbomExport(t *testing.T, img string, update bool) {
	fileImgName := fixtureName(img)
	recording := "testdata/" + fileImgName + "-recording.json"
	require.FileExists(t, recording, "no recording for %s; see README.md", img)

	r := test.NewCliTestRunner("./cnspec", "sbom", "docker", img, "--use-recording", recording)
	err := r.Run()
	require.NoError(t, err)
	assert.Equal(t, 0, r.ExitCode())

	output := string(r.Stdout())
	if update {
		require.NoError(t, os.WriteFile("testdata/"+fileImgName+"-cli.txt", r.Stdout(), 0o600))
	}

	expected, err := os.ReadFile("testdata/" + fileImgName + "-cli.txt")
	require.NoError(t, err)

	if output != string(expected) {
		fmt.Println("stdout:\n", output)
		fmt.Println("stderr:\n", string(r.Stderr()))
	}
	assert.Equal(t, string(expected), output)
	assert.NotEmpty(t, strings.TrimSpace(output), "an empty report would match an empty golden")
}
