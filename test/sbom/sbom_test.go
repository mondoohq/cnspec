// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

//go:build debugtest
// +build debugtest

package sbom

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/mql/v13/test"
)

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

func TestSbomGeneration(t *testing.T) {
	once.Do(setup)

	images := []string{
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

	// test all images sequentially since they use os.stdout
	for i := range images {
		t.Run(images[i], func(t *testing.T) {
			testSbomExport(t, images[i], false, false)
		})
	}
}

func testSbomExport(t *testing.T, img string, update bool, useRecording bool) {
	fileImgName := strings.ReplaceAll(img, ":", "-")
	fileImgName = strings.ReplaceAll(fileImgName, ".", "-")
	fileImgName = strings.ReplaceAll(fileImgName, "/", "-")

	args := []string{"sbom", "docker", img}
	if useRecording {
		args = append(args, "--use-recording", "testdata/"+fileImgName+"-recording.json")
	}

	r := test.NewCliTestRunner("./cnspec", args...)
	err := r.Run()
	require.NoError(t, err)
	assert.Equal(t, 0, r.ExitCode())

	// Check the output
	fmt.Println("stdout:\n", string(r.Stdout()))
	fmt.Println("stderr:\n", string(r.Stderr()))

	if update {
		os.WriteFile("testdata/"+fileImgName+"-cli.txt", r.Stdout(), 0600)
	}

	expected, err := os.ReadFile("testdata/" + fileImgName + "-cli.txt")
	require.NoError(t, err)

	output := string(r.Stdout())
	assert.Equal(t, string(expected), output)
	assert.NotEmpty(t, strings.TrimSpace(output))
}
