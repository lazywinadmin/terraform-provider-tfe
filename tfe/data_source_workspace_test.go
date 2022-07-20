package tfe

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccTFEWorkspaceDataSource_remoteStateConsumers(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	rInt1 := r.Int()
	rInt2 := r.Int()

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccTFEWorkspaceDataSourceConfig_remoteStateConsumers(rInt1, rInt2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.tfe_workspace.foobar", "id"),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "name", fmt.Sprintf("workspace-test-%d", rInt1)),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "global_remote_state", "false"),
					testAccCheckTFEWorkspaceDataSourceHasRemoteStateConsumers("data.tfe_workspace.foobar", 1),
				),
			},
		},
	})
}

func testAccCheckTFEWorkspaceDataSourceHasRemoteStateConsumers(dataWorkspace string, idsLen int) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		org, ok := s.RootModule().Resources[dataWorkspace]
		if !ok {
			return fmt.Errorf("Data workspace '%s' not found.", dataWorkspace)
		}
		numRemoteStateConsumersStr := org.Primary.Attributes["remote_state_consumer_ids.#"]
		numRemoteStateConsumers, _ := strconv.Atoi(numRemoteStateConsumersStr)

		if numRemoteStateConsumers != idsLen {
			return fmt.Errorf("Expected %d remote_state_consumer_ids, but found %d.", idsLen, numRemoteStateConsumers)
		}

		return nil
	}
}

func TestAccTFEWorkspaceDataSource_basic(t *testing.T) {
	rInt := rand.New(rand.NewSource(time.Now().UnixNano())).Int()
	orgName := fmt.Sprintf("tst-terraform-%d", rInt)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccTFEWorkspaceDataSourceConfig(rInt),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.tfe_workspace.foobar", "id"),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "name", fmt.Sprintf("workspace-test-%d", rInt)),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "organization", orgName),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "description", "provider-testing"),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "global_remote_state", "true"),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "allow_destroy_plan", "false"),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "auto_apply", "true"),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "file_triggers_enabled", "true"),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "policy_check_failures", "0"),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "queue_all_runs", "false"),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "resource_count", "0"),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "run_failures", "0"),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "runs_count", "0"),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "speculative_enabled", "true"),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "structured_run_output_enabled", "true"),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "tag_names.0", "modules"),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "tag_names.1", "shared"),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "terraform_version", "0.11.1"),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "trigger_prefixes.#", "2"),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "trigger_prefixes.0", "/modules"),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "trigger_prefixes.1", "/shared"),
					resource.TestCheckResourceAttr(
						"data.tfe_workspace.foobar", "working_directory", "terraform/test"),
				),
			},
		},
	})
}

func testAccTFEWorkspaceDataSourceConfig(rInt int) string {
	return fmt.Sprintf(`
resource "tfe_organization" "foobar" {
  name  = "tst-terraform-%d"
  email = "admin@company.com"
}

resource "tfe_workspace" "foobar" {
  name                  = "workspace-test-%d"
  organization          = tfe_organization.foobar.id
  description           = "provider-testing"
  allow_destroy_plan    = false
  auto_apply            = true
  file_triggers_enabled = true
  queue_all_runs        = false
  speculative_enabled   = true
  tag_names             = ["modules", "shared"]
  terraform_version     = "0.11.1"
  trigger_prefixes      = ["/modules", "/shared"]
  working_directory     = "terraform/test"
  global_remote_state   = true
}

data "tfe_workspace" "foobar" {
  name         = tfe_workspace.foobar.name
  organization = tfe_workspace.foobar.organization
  depends_on   = [tfe_workspace.foobar]
}`, rInt, rInt)
}

func testAccTFEWorkspaceDataSourceConfig_remoteStateConsumers(rInt1, rInt2 int) string {
	return fmt.Sprintf(`
resource "tfe_organization" "foobar" {
  name  = "tst-terraform-%d"
  email = "admin@company.com"
}

resource "tfe_workspace" "buzz" {
  name                      = "workspace-test-%d"
  organization              = tfe_organization.foobar.id
}

resource "tfe_workspace" "foobar" {
  name                      = "workspace-test-%d"
  organization              = tfe_organization.foobar.id
	global_remote_state       = false
	remote_state_consumer_ids = [resource.tfe_workspace.buzz.id]
}

data "tfe_workspace" "foobar" {
  name         = tfe_workspace.foobar.name
  organization = tfe_workspace.foobar.organization
	depends_on   = [tfe_workspace.foobar]
}`, rInt1, rInt2, rInt1)
}
