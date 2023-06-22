package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccExecTestDataSource(t *testing.T) {
	img, err := remote.Image(name.MustParseReference("cgr.dev/chainguard/wolfi-base:latest"))
	if err != nil {
		t.Fatalf("failed to fetch image: %v", err)
	}
	d, err := img.Digest()
	if err != nil {
		t.Fatalf("failed to get image digest: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: fmt.Sprintf(`data "oci_exec_test" "test" {
  digest = "cgr.dev/chainguard/wolfi-base@%s"

  script = "docker run --rm $${IMAGE_NAME} echo hello | grep hello"
}`, d.String()),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.oci_exec_test.test", "digest", fmt.Sprintf("cgr.dev/chainguard/wolfi-base@%s", d.String())),
				resource.TestCheckResourceAttr("data.oci_exec_test.test", "id", fmt.Sprintf("cgr.dev/chainguard/wolfi-base@%s", d.String())),
				resource.TestCheckResourceAttr("data.oci_exec_test.test", "exit_code", "0"),
				resource.TestMatchResourceAttr("data.oci_exec_test.test", "output", regexp.MustCompile("hello\n")),
			),
		}, {
			Config: fmt.Sprintf(`data "oci_exec_test" "script-test" {
				digest = "cgr.dev/chainguard/wolfi-base@%s"

				script = "${path.module}/testdata/test.sh"
			  }`, d.String()),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.oci_exec_test.script-test", "digest", fmt.Sprintf("cgr.dev/chainguard/wolfi-base@%s", d.String())),
				resource.TestCheckResourceAttr("data.oci_exec_test.script-test", "id", fmt.Sprintf("cgr.dev/chainguard/wolfi-base@%s", d.String())),
				resource.TestCheckResourceAttr("data.oci_exec_test.script-test", "exit_code", "0"),
				resource.TestMatchResourceAttr("data.oci_exec_test.script-test", "output", regexp.MustCompile("hello\n")),
			),
		}, {
			Config: fmt.Sprintf(`data "oci_exec_test" "env" {
  digest = "cgr.dev/chainguard/wolfi-base@%s"

  env {
	name  = "FOO"
	value = "bar"
  }
  env {
	name  = "BAR"
	value = "baz"
  }

  script = "echo IMAGE_NAME=$${IMAGE_NAME} IMAGE_REPOSITORY=$${IMAGE_REPOSITORY} IMAGE_REGISTRY=$${IMAGE_REGISTRY} FOO=bar BAR=baz FREE_PORT=$${FREE_PORT}"
}`, d.String()),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.oci_exec_test.env", "digest", fmt.Sprintf("cgr.dev/chainguard/wolfi-base@%s", d.String())),
				resource.TestCheckResourceAttr("data.oci_exec_test.env", "id", fmt.Sprintf("cgr.dev/chainguard/wolfi-base@%s", d.String())),
				resource.TestCheckResourceAttr("data.oci_exec_test.env", "exit_code", "0"),
				resource.TestMatchResourceAttr("data.oci_exec_test.env", "output", regexp.MustCompile(fmt.Sprintf("IMAGE_NAME=cgr.dev/chainguard/wolfi-base@%s IMAGE_REPOSITORY=chainguard/wolfi-base IMAGE_REGISTRY=cgr.dev FOO=bar BAR=baz FREE_PORT=[0-9]+\n", d.String()))),
			),
		}, {
			Config: fmt.Sprintf(`data "oci_exec_test" "fail" {
  digest = "cgr.dev/chainguard/wolfi-base@%s"

  script = "echo failed && exit 12"
}`, d.String()),
			ExpectError: regexp.MustCompile(`Test failed for ref\ncgr.dev/chainguard/wolfi-base@sha256:[0-9a-f]{64},\ngot error: exit status 12\nfailed`),
			// We don't get the exit code or output because the datasource failed.
		}, {
			Config: fmt.Sprintf(`data "oci_exec_test" "timeout" {
	  digest = "cgr.dev/chainguard/wolfi-base@%s"
	  timeout_seconds = 1

	  script = "sleep 6"
	}`, d.String()),
			ExpectError: regexp.MustCompile(`Test for ref\ncgr.dev/chainguard/wolfi-base@sha256:[0-9a-f]{64}\ntimed out after 1 seconds`),
		}, {
			Config: fmt.Sprintf(`data "oci_exec_test" "working_dir" {
		  digest = "cgr.dev/chainguard/wolfi-base@%s"
		  working_dir = "${path.module}/../../"

		  script = "grep 'Terraform Provider for OCI operations' README.md"
		}`, d.String()),
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.oci_exec_test.working_dir", "digest", fmt.Sprintf("cgr.dev/chainguard/wolfi-base@%s", d.String())),
				resource.TestCheckResourceAttr("data.oci_exec_test.working_dir", "id", fmt.Sprintf("cgr.dev/chainguard/wolfi-base@%s", d.String())),
				resource.TestCheckResourceAttr("data.oci_exec_test.working_dir", "exit_code", "0"),
				resource.TestCheckResourceAttr("data.oci_exec_test.working_dir", "output", "# Terraform Provider for OCI operations\n"),
			),
		}},
	})

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"oci": providerserver.NewProtocol6WithError(&OCIProvider{
				defaultExecTimeoutSeconds: 1,
			}),
		}, Steps: []resource.TestStep{{
			Config: fmt.Sprintf(`data "oci_exec_test" "provider-timeout" {
  digest = "cgr.dev/chainguard/wolfi-base@%s"

  script = "sleep 6"
}`, d.String()),
			ExpectError: regexp.MustCompile(`Test for ref\ncgr.dev/chainguard/wolfi-base@sha256:[0-9a-f]{64}\ntimed out after 1 seconds`),
		}},
	})

}

// TestAccExecTestDataSource_Background tests a script that starts the container in
// the background, and `docker rm`s it.
func TestAccExecTestDataSource_Background(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `data "oci_exec_test" "bg-test" {
	digest = "cgr.dev/chainguard-private/hubble-ui@sha256:1412f6ce08e130fc293978f9ed79ba4818ffcd8fe7b0277b7e62e7f8b45d1b9a"

	script = "${path.module}/testdata/bg-test.sh"
  }`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr("data.oci_exec_test.bg-test", "digest", "cgr.dev/chainguard-private/hubble-ui@sha256:1412f6ce08e130fc293978f9ed79ba4818ffcd8fe7b0277b7e62e7f8b45d1b9a"),
				resource.TestCheckResourceAttr("data.oci_exec_test.bg-test", "id", "cgr.dev/chainguard-private/hubble-ui@sha256:1412f6ce08e130fc293978f9ed79ba4818ffcd8fe7b0277b7e62e7f8b45d1b9a"),
				resource.TestCheckResourceAttr("data.oci_exec_test.bg-test", "exit_code", "0"),
			),
		}},
	})
}
