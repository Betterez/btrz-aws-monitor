package btrzaws

import "testing"

type AwsTestInstance struct {
	BetterezInstance
}

func createTestInstance() *AwsTestInstance {
	result := &AwsTestInstance{
	// BetterezInstanceEnvironment:       "production",
	// BetterezInstance.BuildNumber:      1,
	// BetterezInstance.InstanceID:       "12345",
	// BetterezInstance.InstanceName:     "test1",
	//PrivateIPAddress: "localhost",
	}
	result.Environment = "production"
	return result
}

func TestHealthcheck(t *testing.T) {
	const expectedResult = "http://localhost/healthcheck"
	testInstance := createTestInstance()
	if testInstance == nil {
		t.Fatal("failed to create a test instance")
	}
}
