package tfe

import (
	"fmt"
	"log"
	"strings"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceTFEOrganization() *schema.Resource {
	return &schema.Resource{
		Create: resourceTFEOrganizationCreate,
		Read:   resourceTFEOrganizationRead,
		Update: resourceTFEOrganizationUpdate,
		Delete: resourceTFEOrganizationDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return strings.EqualFold(old, new)
				},
			},

			"email": {
				Type:     schema.TypeString,
				Required: true,
			},

			"session_timeout_minutes": {
				Type:     schema.TypeInt,
				Optional: true,
			},

			"session_remember_minutes": {
				Type:     schema.TypeInt,
				Optional: true,
			},

			"collaborator_auth_policy": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  string(tfe.AuthPolicyPassword),
				ValidateFunc: validation.StringInSlice(
					[]string{
						string(tfe.AuthPolicyPassword),
						string(tfe.AuthPolicyTwoFactor),
					},
					false,
				),
			},

			"owners_team_saml_role_id": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"cost_estimation_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"send_passing_statuses_for_untriggered_speculative_plans": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"admin_settings": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"workspace_limit": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourceTFEOrganizationCreate(d *schema.ResourceData, meta interface{}) error {
	tfeClient := meta.(*tfe.Client)

	// Get the organization name.
	name := d.Get("name").(string)

	// Create a new options struct.
	options := tfe.OrganizationCreateOptions{
		Name:  tfe.String(name),
		Email: tfe.String(d.Get("email").(string)),
	}

	log.Printf("[DEBUG] Create new organization: %s", name)
	org, err := tfeClient.Organizations.Create(ctx, options)
	if err != nil {
		return fmt.Errorf("Error creating the new organization %s: %w", name, err)
	}

	d.SetId(org.Name)

	return resourceTFEOrganizationUpdate(d, meta)
}

func resourceTFEOrganizationRead(d *schema.ResourceData, meta interface{}) error {
	tfeClient := meta.(*tfe.Client)

	log.Printf("[DEBUG] Read configuration of organization: %s", d.Id())
	org, err := tfeClient.Organizations.Read(ctx, d.Id())
	if err != nil {
		if err == tfe.ErrResourceNotFound {
			log.Printf("[DEBUG] Organization %s does no longer exist", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	// Update the config.
	d.Set("name", org.Name)
	d.Set("email", org.Email)
	d.Set("session_timeout_minutes", org.SessionTimeout)
	d.Set("session_remember_minutes", org.SessionRemember)
	d.Set("collaborator_auth_policy", org.CollaboratorAuthPolicy)
	d.Set("owners_team_saml_role_id", org.OwnersTeamSAMLRoleID)
	d.Set("cost_estimation_enabled", org.CostEstimationEnabled)
	d.Set("send_passing_statuses_for_untriggered_speculative_plans", org.SendPassingStatusesForUntriggeredSpeculativePlans)

	// update admin settings
	log.Printf("[DEBUG] Read admin settings of organization: %s", d.Id())
	adminOrg, err := tfeClient.Admin.Organizations.Read(ctx, d.Id())
	if err != nil {
		return err
	}
	var adminSettings []interface{}

	if adminOrg.WorkspaceLimit != nil && *adminOrg.WorkspaceLimit != 0 {
		adminSettings = append(adminSettings, map[string]interface{}{
			"workspace_limit": *adminOrg.WorkspaceLimit,
		})
	}
	if err := d.Set("admin_settings", adminSettings); err != nil {
		return fmt.Errorf("error setting admin settings for organization %s: %w", d.Id(), err)
	}

	return nil
}

func resourceTFEOrganizationUpdate(d *schema.ResourceData, meta interface{}) error {
	tfeClient := meta.(*tfe.Client)

	// Create a new options struct.
	options := tfe.OrganizationUpdateOptions{
		Name:  tfe.String(d.Get("name").(string)),
		Email: tfe.String(d.Get("email").(string)),
	}

	// If session_timeout is supplied, set it using the options struct.
	if sessionTimeout, ok := d.GetOk("session_timeout_minutes"); ok {
		options.SessionTimeout = tfe.Int(sessionTimeout.(int))
	}

	// If session_remember is supplied, set it using the options struct.
	if sessionRemember, ok := d.GetOk("session_remember_minutes"); ok {
		options.SessionRemember = tfe.Int(sessionRemember.(int))
	}

	// If collaborator_auth_policy is supplied, set it using the options struct.
	if authPolicy, ok := d.GetOk("collaborator_auth_policy"); ok {
		options.CollaboratorAuthPolicy = tfe.AuthPolicy(tfe.AuthPolicyType(authPolicy.(string)))
	}

	// If owners_team_saml_role_id is supplied, set it using the options struct.
	if ownersTeamSAMLRoleID, ok := d.GetOk("owners_team_saml_role_id"); ok {
		options.OwnersTeamSAMLRoleID = tfe.String(ownersTeamSAMLRoleID.(string))
	}

	// If cost_estimation_enabled is supplied, set it using the options struct.
	if costEstimationEnabled, ok := d.GetOkExists("cost_estimation_enabled"); ok {
		options.CostEstimationEnabled = tfe.Bool(costEstimationEnabled.(bool))
	}

	// If send_passing_statuses_for_untriggered_speculative_plans is supplied, set it using the options struct.
	if sendPassingStatusesForUntriggeredSpeculativePlans, ok := d.GetOk("send_passing_statuses_for_untriggered_speculative_plans"); ok {
		options.SendPassingStatusesForUntriggeredSpeculativePlans = tfe.Bool(sendPassingStatusesForUntriggeredSpeculativePlans.(bool))
	}

	// update workspace limit setting only if it has changed
	if d.HasChange("admin_settings.0.workspace_limit") {
		adminOpts := tfe.AdminOrganizationUpdateOptions{}

		newLimit := d.Get("admin_settings.0.workspace_limit").(int)
		adminOpts.WorkspaceLimit = &newLimit

		_, err := tfeClient.Admin.Organizations.Update(ctx, d.Id(), adminOpts)
		if err != nil {
			return fmt.Errorf("Error updating admin settings for organization %s: %w", d.Id(), err)
		}
	}

	log.Printf("[DEBUG] Update configuration of organization: %s", d.Id())
	org, err := tfeClient.Organizations.Update(ctx, d.Id(), options)
	if err != nil {
		return fmt.Errorf("Error updating organization %s: %w", d.Id(), err)
	}

	d.SetId(org.Name)

	return resourceTFEOrganizationRead(d, meta)
}

func resourceTFEOrganizationDelete(d *schema.ResourceData, meta interface{}) error {
	tfeClient := meta.(*tfe.Client)

	log.Printf("[DEBUG] Delete organization: %s", d.Id())
	err := tfeClient.Organizations.Delete(ctx, d.Id())
	if err != nil {
		if err == tfe.ErrResourceNotFound {
			return nil
		}
		return fmt.Errorf("Error deleting organization %s: %w", d.Id(), err)
	}

	return nil
}
