package test

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/azure"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

var subscriptionID string = "aa8bf277-fdd4-4ec4-bcd4-3458ddb8af6c"

func TestAzureLinuxVMCreation(t *testing.T) {
	terraformOptions := &terraform.Options{
		TerraformDir: "../",
		Vars: map[string]interface{}{
			"labelPrefix": "parm0100",
		},
	}

	// Ensure Terraform resources are destroyed at the end
	defer terraform.Destroy(t, terraformOptions)

	// Run Terraform Init and Apply
	terraform.InitAndApply(t, terraformOptions)

	// Retrieve output variables from Terraform
	vmName := terraform.Output(t, terraformOptions, "vm_name")
	resourceGroupName := terraform.Output(t, terraformOptions, "resource_group_name")
	publicIPName := terraform.Output(t, terraformOptions, "public_ip_name")
	nsgName := terraform.Output(t, terraformOptions, "nsg_name")

	// Confirm that the Virtual Machine exists
	vmExists := azure.VirtualMachineExists(t, subscriptionID, resourceGroupName, vmName)
	assert.True(t, vmExists, "Virtual Machine does not exist")

	// Check if the Public IP exists
	publicIPExists := azure.PublicIPExists(t, subscriptionID, resourceGroupName, publicIPName)
	assert.True(t, publicIPExists, "Public IP does not exist")

	// Check if the VM is running
	vmPowerState := azure.GetVirtualMachinePowerState(t, subscriptionID, resourceGroupName, vmName)
	assert.Equal(t, "VM running", vmPowerState, "Virtual Machine is not running")

	// Validate Network Security Group existence
	nsgExists := azure.NetworkSecurityGroupExists(t, subscriptionID, resourceGroupName, nsgName)
	assert.True(t, nsgExists, "Network Security Group does not exist")

	// Verify NSG rules allow SSH (Port 22) and HTTP (Port 80)
	nsgRules := azure.GetNetworkSecurityGroupRules(t, subscriptionID, resourceGroupName, nsgName)

	hasSSHRule := false
	hasHTTPRule := false

	for _, rule := range nsgRules {
		if rule.DestinationPortRange == "22" && rule.Access == "Allow" {
			hasSSHRule = true
		}
		if rule.DestinationPortRange == "80" && rule.Access == "Allow" {
			hasHTTPRule = true
		}
	}

	assert.True(t, hasSSHRule, "NSG does not have SSH rule (Port 22 open)")
	assert.True(t, hasHTTPRule, "NSG does not have HTTP rule (Port 80 open)")
}
