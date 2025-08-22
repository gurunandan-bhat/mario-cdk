package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2integrations"
	"github.com/aws/aws-cdk-go/awscdk/v2/awscognito"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsdynamodb"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssecretsmanager"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type MarioCdkStackProps struct {
	awscdk.StackProps
}

func NewMarioCdkStack(scope constructs.Construct, id string, props *MarioCdkStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	// Create a simple lambda to make sure it can read secrets
	marioSecret := awssecretsmanager.Secret_FromSecretCompleteArn(
		stack,
		jsii.String("mario/defaultSecret"),
		jsii.String("arn:aws:secretsmanager:ap-south-1:566275025856:secret:mario/defaultSecret-NI1lQX"),
	)
	testLambda := awslambda.NewFunction(stack, jsii.String("MarioTestLambda"), &awslambda.FunctionProps{
		FunctionName: jsii.String("MarioTestLambda"),
		Runtime:      awslambda.Runtime_PROVIDED_AL2023(),
		Code:         awslambda.AssetCode_FromAsset(jsii.String("lambda/secrets/function.zip"), nil),
		Handler:      jsii.String("main"),
	})
	marioSecret.GrantRead(testLambda, nil)
	testLambda.ApplyRemovalPolicy(awscdk.RemovalPolicy_DESTROY)

	testAPI := awsapigatewayv2.NewHttpApi(stack, jsii.String("TestAPI"), &awsapigatewayv2.HttpApiProps{
		ApiName: jsii.String("TestAPI"),
		CorsPreflight: &awsapigatewayv2.CorsPreflightOptions{
			AllowHeaders: jsii.Strings("Authorization"),
			AllowMethods: &[]awsapigatewayv2.CorsHttpMethod{
				awsapigatewayv2.CorsHttpMethod_GET,
				awsapigatewayv2.CorsHttpMethod_POST,
				awsapigatewayv2.CorsHttpMethod_OPTIONS,
				awsapigatewayv2.CorsHttpMethod_HEAD,
			},
			AllowOrigins: jsii.Strings("*"),
		},
		// DefaultAuthorizer:          marioAuth,
		// DefaultAuthorizationScopes: jsii.Strings("openid", "email"),
	})
	testAPI.ApplyRemovalPolicy(awscdk.RemovalPolicy_DESTROY)
	testIntegration := awsapigatewayv2integrations.NewHttpLambdaIntegration(
		jsii.String("TestIntegration"),
		testLambda,
		&awsapigatewayv2integrations.HttpLambdaIntegrationProps{},
	)
	testAPI.AddRoutes(&awsapigatewayv2.AddRoutesOptions{
		Path:        jsii.String("/secret/{id}"),
		Methods:     &[]awsapigatewayv2.HttpMethod{awsapigatewayv2.HttpMethod_GET},
		Integration: testIntegration,
	})

	marioAuthLogTable := awsdynamodb.NewTable(stack, jsii.String("MarioAuthLog"), &awsdynamodb.TableProps{
		TableName: jsii.String("MarioAuthLog"),
		PartitionKey: &awsdynamodb.Attribute{
			Name: jsii.String("PK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		SortKey: &awsdynamodb.Attribute{
			Name: jsii.String("SK"),
			Type: awsdynamodb.AttributeType_STRING,
		},
		BillingMode:   awsdynamodb.BillingMode_PAY_PER_REQUEST,
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})
	marioAuthLogTrigger := awslambda.NewFunction(stack, jsii.String("MarioAuthLogTrigger"), &awslambda.FunctionProps{
		FunctionName: jsii.String("MarioAuthLogTrigger"),
		Runtime:      awslambda.Runtime_PROVIDED_AL2023(),
		Code:         awslambda.AssetCode_FromAsset(jsii.String("lambda/users/function.zip"), nil),
		Handler:      jsii.String("main"),
	})
	marioAuthLogTrigger.ApplyRemovalPolicy(awscdk.RemovalPolicy_DESTROY)
	marioAuthLogTrigger.AddEnvironment(jsii.String("AUTHLOG_TABLENAME"), marioAuthLogTable.TableName(), nil)
	marioAuthLogTable.GrantFullAccess(marioAuthLogTrigger)

	// Create a User Pool and client
	marioUserPool := awscognito.NewUserPool(stack, jsii.String("MarioUserPool"), &awscognito.UserPoolProps{
		UserPoolName:    jsii.String("MarioUserPool"),
		AccountRecovery: awscognito.AccountRecovery_EMAIL_ONLY,
		LambdaTriggers: &awscognito.UserPoolTriggers{
			PostConfirmation:   marioAuthLogTrigger,
			PostAuthentication: marioAuthLogTrigger,
		},
		SelfSignUpEnabled: jsii.Bool(true),
		PasswordPolicy: &awscognito.PasswordPolicy{
			MinLength:        jsii.Number(8),
			RequireLowercase: jsii.Bool(true),
			RequireUppercase: jsii.Bool(true),
			RequireDigits:    jsii.Bool(true),
			RequireSymbols:   jsii.Bool(true),
		},
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
		StandardAttributes: &awscognito.StandardAttributes{
			Fullname: &awscognito.StandardAttribute{
				Required: jsii.Bool(true),
			},
			Email: &awscognito.StandardAttribute{
				Required: jsii.Bool(true),
			},
		},
		SignInAliases: &awscognito.SignInAliases{
			Email:    jsii.Bool(true),
			Username: jsii.Bool(false),
		},
	})
	marioUserPoolClient := marioUserPool.AddClient(jsii.String("MarioUserPoolClient"), &awscognito.UserPoolClientOptions{
		UserPoolClientName:    jsii.String("MarioUserPoolClient"),
		EnableTokenRevocation: jsii.Bool(true),
		GenerateSecret:        jsii.Bool(true),
		AuthFlows: &awscognito.AuthFlow{
			UserSrp: jsii.Bool(true),
		},
		OAuth: &awscognito.OAuthSettings{
			CallbackUrls: jsii.Strings("http://localhost:2000/callback"),
			Flows: &awscognito.OAuthFlows{
				AuthorizationCodeGrant: jsii.Bool(true),
			},
			LogoutUrls: jsii.Strings("http://localhost:2000/logout"),
			Scopes: &[]awscognito.OAuthScope{
				awscognito.OAuthScope_OPENID(),
				awscognito.OAuthScope_EMAIL(),
				awscognito.OAuthScope_PROFILE(),
			},
		},
		WriteAttributes: (awscognito.NewClientAttributes()).WithStandardAttributes(&awscognito.StandardAttributesMask{
			Fullname: jsii.Bool(true),
			Email:    jsii.Bool(true),
		}),
	})
	marioUserPoolClient.ApplyRemovalPolicy(awscdk.RemovalPolicy_DESTROY)
	awscognito.NewCfnManagedLoginBranding(stack, jsii.String("MarioLoginBranding"), &awscognito.CfnManagedLoginBrandingProps{
		UserPoolId:               marioUserPool.UserPoolId(),
		ClientId:                 marioUserPoolClient.UserPoolClientId(),
		UseCognitoProvidedValues: jsii.Bool(true),
	})
	marioUserPool.AddDomain(
		jsii.String("MarioUserPoolDomain"),
		&awscognito.UserPoolDomainOptions{
			CognitoDomain: &awscognito.CognitoDomainOptions{
				DomainPrefix: jsii.String("mario"),
			},
			ManagedLoginVersion: awscognito.ManagedLoginVersion_NEWER_MANAGED_LOGIN,
		},
	)

	region := marioUserPool.Stack().Region()
	poolID := marioUserPool.UserPoolId()
	issuer := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s", *region, *poolID)
	clientID := marioUserPoolClient.UserPoolClientId()
	clientSecret := marioUserPoolClient.UserPoolClientSecret().UnsafeUnwrap()

	awssecretsmanager.NewSecret(stack, jsii.String("MarioUserPoolSecret"), &awssecretsmanager.SecretProps{
		SecretName: jsii.String("MarioUserPoolSecret"),
		SecretObjectValue: &map[string]awscdk.SecretValue{
			"issuerURL":    awscdk.SecretValue_UnsafePlainText(jsii.String(issuer)),
			"clientID":     awscdk.SecretValue_UnsafePlainText(clientID),
			"clientSecret": awscdk.SecretValue_UnsafePlainText(clientSecret),
		},
	})

	awscdk.NewCfnOutput(stack, jsii.String("Auth Trigger Role"), &awscdk.CfnOutputProps{
		Value:       marioAuthLogTrigger.Role().RoleName(),
		Description: jsii.String("Role that runs the auth trigger"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("testAPI URL"), &awscdk.CfnOutputProps{
		Value:       testAPI.Url(),
		Description: jsii.String("The URL to test"),
	})

	awscdk.NewCfnOutput(stack, jsii.String("mario Auth Log Table"), &awscdk.CfnOutputProps{
		Value:       marioAuthLogTable.TableName(),
		Description: jsii.String("The name of the table"),
	})

	return stack
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	NewMarioCdkStack(app, "MarioCdkStack", &MarioCdkStackProps{
		awscdk.StackProps{
			Env:       env(),
			StackName: jsii.String("MarioCdkStack"),
		},
	})

	app.Synth(nil)
}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	// If unspecified, this stack will be "environment-agnostic".
	// Account/Region-dependent features and context lookups will not work, but a
	// single synthesized template can be deployed anywhere.
	//---------------------------------------------------------------------------
	// return nil

	// Uncomment if you know exactly what account and region you want to deploy
	// the stack to. This is the recommendation for production stacks.
	//---------------------------------------------------------------------------
	// return &awscdk.Environment{
	//  Account: jsii.String("123456789012"),
	//  Region:  jsii.String("us-east-1"),
	// }

	// Uncomment to specialize this stack for the AWS Account and Region that are
	// implied by the current CLI configuration. This is recommended for dev
	// stacks.
	//---------------------------------------------------------------------------
	return &awscdk.Environment{
		Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
		Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	}
}
