package k8s

import (
	"encoding/json"
	"os"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
)

func TestCheckDeploymentAvailability(t *testing.T) {
	testCases := map[string]struct {
		DeploymentJsonPath  string
		IsExpectedAvailable bool
	}{
		"available deployment": {
			DeploymentJsonPath:  "./testdata/deployments/snmp-available.json",
			IsExpectedAvailable: true,
		},
		"currently restarting": {
			DeploymentJsonPath:  "./testdata/deployments/snmp-restarting.json",
			IsExpectedAvailable: false,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			var deployment appsv1.Deployment
			deploymentJsonBytes, err := os.ReadFile(tc.DeploymentJsonPath)
			if err != nil {
				t.Fatalf("%s does not exist.\n", tc.DeploymentJsonPath)
			}

			err = json.Unmarshal(deploymentJsonBytes, &deployment)
			if err != nil {
				t.Fatalf("could not unmarshall %s\n", tc.DeploymentJsonPath)
			}

			isDeploymentAvailable := CheckDeploymentAvailability(deployment, tc.GracePeriod)
			if isDeploymentAvailable != tc.IsExpectedAvailable {
				t.Fatalf("availability mismatch expected=%t, got=%t", tc.IsExpectedAvailable, isDeploymentAvailable)
			}
		})
	}
}
