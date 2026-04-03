package main

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pulumi/pulumi-docker/sdk/v4/go/docker"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/apigateway"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/artifactregistry"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudrunv2"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/organizations"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		gcpCfg := config.New(ctx, "gcp")
		appCfg := config.New(ctx, ctx.Project())
		project := gcpCfg.Require("project")
		region := gcpCfg.Require("region")
		repositoryID := appCfg.Get("artifactRegistryRepository")
		if repositoryID == "" {
			repositoryID = "temperature-images"
		}
		imageName := appCfg.Get("imageName")
		if imageName == "" {
			imageName = "temperature-service"
		}
		imageTag := appCfg.Get("imageTag")
		if imageTag == "" {
			imageTag = "v1"
		}
		serviceName := appCfg.Get("serviceName")
		if serviceName == "" {
			serviceName = "temperature-service"
		}
		apiID := normalizeIdentifier(serviceName+"-api", 63)
		gatewayID := normalizeIdentifier(serviceName+"-gateway", 63)
		gatewayServiceAccountID := normalizeIdentifier(serviceName+"-gateway", 30)

		services := []string{
			"apigateway.googleapis.com",
			"artifactregistry.googleapis.com",
			"run.googleapis.com",
			"servicecontrol.googleapis.com",
			"servicemanagement.googleapis.com",
		}
		enabledAPIs := make([]pulumi.Resource, 0, len(services))
		for _, api := range services {
			name := strings.ReplaceAll(strings.TrimSuffix(api, ".googleapis.com"), ".", "-")
			service, err := projects.NewService(ctx, name, &projects.ServiceArgs{
				Project:                         pulumi.String(project),
				Service:                         pulumi.String(api),
				DisableOnDestroy:                pulumi.Bool(false),
				CheckIfServiceHasUsageOnDestroy: pulumi.Bool(false),
			})
			if err != nil {
				return err
			}
			enabledAPIs = append(enabledAPIs, service)
		}

		repo, err := artifactregistry.NewRepository(ctx, "images", &artifactregistry.RepositoryArgs{
			Project:      pulumi.String(project),
			Location:     pulumi.String(region),
			RepositoryId: pulumi.String(repositoryID),
			Description:  pulumi.String("Docker images for the temperature Cloud Run example"),
			Format:       pulumi.String("DOCKER"),
		}, pulumi.DependsOn(enabledAPIs))
		if err != nil {
			return err
		}

		clientConfig := organizations.GetClientConfigOutput(ctx)
		serverImageRef := pulumi.All(repo.RegistryUri, pulumi.String(imageName), pulumi.String(imageTag)).ApplyT(func(values []any) string {
			return fmt.Sprintf("%s/%s:%s", values[0].(string), values[1].(string), values[2].(string))
		}).(pulumi.StringOutput)

		rootDir := ctx.RootDirectory()
		dockerfile := pulumi.String(filepath.Join(rootDir, "Dockerfile"))
		contextDir := pulumi.String(rootDir)

		image, err := docker.NewImage(ctx, "server-image", &docker.ImageArgs{
			ImageName: serverImageRef,
			Build: &docker.DockerBuildArgs{
				Context:    contextDir,
				Dockerfile: dockerfile,
				Platform:   pulumi.String("linux/amd64"),
			},
			Registry: &docker.RegistryArgs{
				Server:   repo.RegistryUri,
				Username: pulumi.String("oauth2accesstoken"),
				Password: clientConfig.AccessToken(),
			},
		}, pulumi.DependsOn([]pulumi.Resource{repo}))
		if err != nil {
			return err
		}

		service, err := cloudrunv2.NewService(ctx, "service", &cloudrunv2.ServiceArgs{
			Project:            pulumi.String(project),
			Name:               pulumi.String(serviceName),
			Location:           pulumi.String(region),
			DeletionProtection: pulumi.Bool(false),
			Ingress:            pulumi.String("INGRESS_TRAFFIC_ALL"),
			InvokerIamDisabled: pulumi.Bool(false),
			Description:        pulumi.String("Authenticated Cloud Run service serving a tiny temperature API behind API Gateway"),
			Template: &cloudrunv2.ServiceTemplateArgs{
				ExecutionEnvironment:          pulumi.String("EXECUTION_ENVIRONMENT_GEN2"),
				MaxInstanceRequestConcurrency: pulumi.Int(80),
				Timeout:                       pulumi.String("15s"),
				Scaling: &cloudrunv2.ServiceTemplateScalingArgs{
					MinInstanceCount: pulumi.Int(0),
					MaxInstanceCount: pulumi.Int(3),
				},
				Containers: cloudrunv2.ServiceTemplateContainerArray{
					&cloudrunv2.ServiceTemplateContainerArgs{
						Image: image.RepoDigest,
						Ports: &cloudrunv2.ServiceTemplateContainerPortsArgs{
							ContainerPort: pulumi.Int(8080),
						},
						Resources: &cloudrunv2.ServiceTemplateContainerResourcesArgs{
							CpuIdle: pulumi.Bool(true),
							Limits: pulumi.StringMap{
								"cpu":    pulumi.String("1"),
								"memory": pulumi.String("512Mi"),
							},
							StartupCpuBoost: pulumi.Bool(true),
						},
					},
				},
			},
			Traffics: cloudrunv2.ServiceTrafficArray{
				&cloudrunv2.ServiceTrafficArgs{
					Percent: pulumi.Int(100),
					Type:    pulumi.String("TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"),
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{image}))
		if err != nil {
			return err
		}

		gatewayServiceAccount, err := serviceaccount.NewAccount(ctx, "gateway-service-account", &serviceaccount.AccountArgs{
			Project:     pulumi.String(project),
			AccountId:   pulumi.String(gatewayServiceAccountID),
			DisplayName: pulumi.String("Temperature API Gateway backend"),
			Description: pulumi.String("Service account used by API Gateway to invoke the Cloud Run backend"),
		}, pulumi.DependsOn(enabledAPIs))
		if err != nil {
			return err
		}

		_, err = cloudrunv2.NewServiceIamMember(ctx, "gateway-invoker", &cloudrunv2.ServiceIamMemberArgs{
			Project:  pulumi.String(project),
			Location: pulumi.String(region),
			Name:     service.Name,
			Role:     pulumi.String("roles/run.invoker"),
			Member:   gatewayServiceAccount.Member,
		}, pulumi.DependsOn([]pulumi.Resource{service, gatewayServiceAccount}))
		if err != nil {
			return err
		}

		api, err := apigateway.NewApi(ctx, "service-api", &apigateway.ApiArgs{
			Project:     pulumi.String(project),
			ApiId:       pulumi.String(apiID),
			DisplayName: pulumi.String(serviceName + " API"),
		}, pulumi.DependsOn(enabledAPIs))
		if err != nil {
			return err
		}

		openAPIDocument := service.Uri.ApplyT(func(uri string) string {
			spec := gatewayOpenAPISpec(uri, serviceName+" API")
			return base64.StdEncoding.EncodeToString([]byte(spec))
		}).(pulumi.StringOutput)

		apiConfig, err := apigateway.NewApiConfig(ctx, "service-api-config", &apigateway.ApiConfigArgs{
			Project:           pulumi.String(project),
			Api:               api.ApiId,
			ApiConfigIdPrefix: pulumi.String(normalizeIdentifier(serviceName+"-cfg", 24) + "-"),
			DisplayName:       pulumi.String(serviceName + " config"),
			GatewayConfig: &apigateway.ApiConfigGatewayConfigArgs{
				BackendConfig: &apigateway.ApiConfigGatewayConfigBackendConfigArgs{
					GoogleServiceAccount: gatewayServiceAccount.Email,
				},
			},
			OpenapiDocuments: apigateway.ApiConfigOpenapiDocumentArray{
				&apigateway.ApiConfigOpenapiDocumentArgs{
					Document: &apigateway.ApiConfigOpenapiDocumentDocumentArgs{
						Path:     pulumi.String("openapi.yaml"),
						Contents: openAPIDocument,
					},
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{service, api, gatewayServiceAccount}), pulumi.ReplaceOnChanges([]string{"*"}))
		if err != nil {
			return err
		}

		gateway, err := apigateway.NewGateway(ctx, "service-gateway", &apigateway.GatewayArgs{
			Project:     pulumi.String(project),
			Region:      pulumi.String(region),
			GatewayId:   pulumi.String(gatewayID),
			DisplayName: pulumi.String(serviceName + " gateway"),
			ApiConfig:   apiConfig.Name,
		}, pulumi.DependsOn([]pulumi.Resource{apiConfig}))
		if err != nil {
			return err
		}

		publicURL := gateway.DefaultHostname.ApplyT(func(hostname string) string {
			return fmt.Sprintf("https://%s", hostname)
		}).(pulumi.StringOutput)

		ctx.Export("project", pulumi.String(project))
		ctx.Export("region", pulumi.String(region))
		ctx.Export("artifactRegistry", repo.RegistryUri)
		ctx.Export("imageName", image.ImageName)
		ctx.Export("imageDigest", image.RepoDigest)
		ctx.Export("serviceName", service.Name)
		ctx.Export("cloudRunUri", service.Uri)
		ctx.Export("apiGatewayManagedService", api.ManagedService)
		ctx.Export("apiGatewayServiceAccount", gatewayServiceAccount.Email)
		ctx.Export("gatewayHostname", gateway.DefaultHostname)
		ctx.Export("serviceUrl", publicURL)

		return nil
	})
}

func gatewayOpenAPISpec(serviceURL, title string) string {
	return strings.Join([]string{
		"swagger: '2.0'",
		"info:",
		fmt.Sprintf("  title: %s", title),
		"  description: API Gateway in front of the Cloud Run temperature service.",
		"  version: 1.0.0",
		"schemes:",
		"  - https",
		"produces:",
		"  - application/json",
		"x-google-management:",
		"  metrics:",
		"    - name: requests",
		"      displayName: Requests",
		"      valueType: INT64",
		"      metricKind: DELTA",
		"  quota:",
		"    limits:",
		"      - name: requests-per-minute",
		"        metric: requests",
		"        unit: 1/min/{project}",
		"        values:",
		"          STANDARD: 10",
		"paths:",
		"  /:",
		"    get:",
		"      operationId: getIndex",
		"      x-google-backend:",
		fmt.Sprintf("        address: %s", serviceURL),
		"        path_translation: APPEND_PATH_TO_ADDRESS",
		fmt.Sprintf("        jwt_audience: %s", serviceURL),
		"      x-google-quota:",
		"        metricCosts:",
		"          requests: 1",
		"      security:",
		"        - api_key: []",
		"      responses:",
		"        '200':",
		"          description: OK",
		"  /api/temperature:",
		"    get:",
		"      operationId: getTemperature",
		"      x-google-backend:",
		fmt.Sprintf("        address: %s", serviceURL),
		"        path_translation: APPEND_PATH_TO_ADDRESS",
		fmt.Sprintf("        jwt_audience: %s", serviceURL),
		"      x-google-quota:",
		"        metricCosts:",
		"          requests: 1",
		"      security:",
		"        - api_key: []",
		"      parameters:",
		"        - in: query",
		"          name: lat",
		"          required: true",
		"          type: number",
		"        - in: query",
		"          name: lng",
		"          required: true",
		"          type: number",
		"      responses:",
		"        '200':",
		"          description: OK",
		"securityDefinitions:",
		"  api_key:",
		"    type: apiKey",
		"    name: X-API-Key",
		"    in: header",
	}, "\n")
}

func normalizeIdentifier(value string, maxLen int) string {
	var builder strings.Builder
	lastWasDash := false
	for _, r := range strings.ToLower(value) {
		isAlphaNum := r >= 'a' && r <= 'z' || r >= '0' && r <= '9'
		if isAlphaNum {
			builder.WriteRune(r)
			lastWasDash = false
			continue
		}
		if !lastWasDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastWasDash = true
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		result = "service"
	}
	if result[0] < 'a' || result[0] > 'z' {
		result = "a-" + result
	}
	if len(result) > maxLen {
		result = strings.Trim(result[:maxLen], "-")
	}
	for len(result) < 6 {
		result += "-svc"
		if len(result) > maxLen {
			result = result[:maxLen]
			break
		}
	}
	return strings.Trim(result, "-")
}
