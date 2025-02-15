package test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/azure"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/ssh"
)

var subscriptionID string = "aa8bf277-fdd4-4ec4-bcd4-3458ddb8af6c"

func TestAzureLinuxVMCreation(t *testing.T) {
	terraformOptions := &terraform.Options{
		TerraformDir: "../",
		Vars: map[string]interface{}{
			"labelPrefix": "parm0100",
		},
	}

	defer terraform.Destroy(t, terraformOptions)

	// Run `terraform init` and `terraform apply`. Fail the test if there are any errors.
	terraform.InitAndApply(t, terraformOptions)

	// Run `terraform output` to get the value of output variables
	vmName := terraform.Output(t, terraformOptions, "vm_name")
	resourceGroupName := terraform.Output(t, terraformOptions, "resource_group_name")
	nicID := terraform.Output(t, terraformOptions, "nic_id") // Use NIC ID instead of name
	publicIP := terraform.Output(t, terraformOptions, "public_ip")

	// Confirm VM exists
	assert.True(t, azure.VirtualMachineExists(t, vmName, resourceGroupName, subscriptionID))

	// 1. Confirm NIC exists and is connected to the VM
	vmDetails := azure.GetVirtualMachine(t, subscriptionID, resourceGroupName, vmName)

	// Access network interfaces through NetworkProfile
	networkInterfaces := vmDetails.NetworkProfile.NetworkInterfaces
	nicConnected := false
	for _, nic := range *networkInterfaces { // Dereference pointer
		if strings.Contains(*nic.ID, nicID) {
			nicConnected = true
			break
		}
	}

	assert.True(t, nicConnected, "NIC is not connected to the VM")

	// 2. Confirm the VM is running the correct Ubuntu version
	privateKeyPath := "/path/to/your/private/key"

	// Load private key for SSH authentication
	privateKey, err := os.ReadFile(privateKeyPath)
	if err != nil {
		t.Fatalf("Failed to read private key: %v", err)
	}

	privateKeyParsed, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to parse private key: %v", err)
	}

	// Use the Signer directly for SSH authentication
	sshConfig := &ssh.ClientConfig{
		User:            "ubuntu",
		Auth:            []ssh.AuthMethod{ssh.PublicKey(privateKeyParsed)}, // Correct use of PublicKey from Signer
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect using SSH to the VM
	sshClient, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", publicIP), sshConfig)
	if err != nil {
		t.Fatalf("Failed to create SSH client: %v", err)
	}
	defer sshClient.Close()

	// Run `lsb_release -a` command to check the Ubuntu version
	command := "lsb_release -a"
	session, err := sshClient.NewSession()
	if err != nil {
		t.Fatalf("Failed to create SSH session: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		t.Fatalf("Failed to run SSH command: %v", err)
	}

	expectedVersion := "Ubuntu 20.04 LTS"
	assert.Contains(t, string(output), expectedVersion, fmt.Sprintf("VM is not running the expected Ubuntu version. Found: %s", string(output)))
}
