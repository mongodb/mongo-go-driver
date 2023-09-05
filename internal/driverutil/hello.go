package driverutil

import (
	"os"
	"strings"
)

const AwsLambdaPrefix = "AWS_Lambda_"

const (
	// FaaS environment variable names
	EnvVarAWSExecutionEnv        = "AWS_EXECUTION_ENV"
	EnvVarAWSLambdaRuntimeAPI    = "AWS_LAMBDA_RUNTIME_API"
	EnvVarFunctionsWorkerRuntime = "FUNCTIONS_WORKER_RUNTIME"
	EnvVarKService               = "K_SERVICE"
	EnvVarFunctionName           = "FUNCTION_NAME"
	EnvVarVercel                 = "VERCEL"
)

const (
	// FaaS environment variable names
	EnvVarAWSRegion                   = "AWS_REGION"
	EnvVarAWSLambdaFunctionMemorySize = "AWS_LAMBDA_FUNCTION_MEMORY_SIZE"
	EnvVarFunctionMemoryMB            = "FUNCTION_MEMORY_MB"
	EnvVarFunctionTimeoutSec          = "FUNCTION_TIMEOUT_SEC"
	EnvVarFunctionRegion              = "FUNCTION_REGION"
	EnvVarVercelRegion                = "VERCEL_REGION"
)

const (
	// FaaS environment names used by the client
	EnvNameAWSLambda = "aws.lambda"
	EnvNameAzureFunc = "azure.func"
	EnvNameGCPFunc   = "gcp.func"
	EnvNameVercel    = "vercel"
)

// GetFaasEnvName parses the FaaS environment variable name and returns the
// corresponding name used by the client. If none of the variables or variables
// for multiple names are populated the client.env value MUST be entirely
// omitted. When variables for multiple "client.env.name" values are present,
// "vercel" takes precedence over "aws.lambda"; any other combination MUST cause
// "client.env" to be entirely omitted.
func GetFaasEnvName() string {
	envVars := []string{
		EnvVarAWSExecutionEnv,
		EnvVarAWSLambdaRuntimeAPI,
		EnvVarFunctionsWorkerRuntime,
		EnvVarKService,
		EnvVarFunctionName,
		EnvVarVercel,
	}

	// If none of the variables are populated the client.env value MUST be
	// entirely omitted.
	names := make(map[string]struct{})

	for _, envVar := range envVars {
		val := os.Getenv(envVar)
		if val == "" {
			continue
		}

		var name string

		switch envVar {
		case EnvVarAWSExecutionEnv:
			if !strings.HasPrefix(val, AwsLambdaPrefix) {
				continue
			}

			name = EnvNameAWSLambda
		case EnvVarAWSLambdaRuntimeAPI:
			name = EnvNameAWSLambda
		case EnvVarFunctionsWorkerRuntime:
			name = EnvNameAzureFunc
		case EnvVarKService, EnvVarFunctionName:
			name = EnvNameGCPFunc
		case EnvVarVercel:
			// "vercel" takes precedence over "aws.lambda".
			delete(names, EnvNameAWSLambda)

			name = EnvNameVercel
		}

		names[name] = struct{}{}
		if len(names) > 1 {
			// If multiple names are populated the client.env value
			// MUST be entirely omitted.
			names = nil

			break
		}
	}

	for name := range names {
		return name
	}

	return ""
}
