package main

import (
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2integrations"
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

	// The code that defines your stack goes here
	loginLambda := awslambda.NewFunction(stack, jsii.String("MarioCDKLambdaLoginURL"), &awslambda.FunctionProps{
		Runtime: awslambda.Runtime_PROVIDED_AL2023(),
		Code:    awslambda.AssetCode_FromAsset(jsii.String("lambda/secrets/function.zip"), nil),
		Handler: jsii.String("main"),
	})
	loginLambda.ApplyRemovalPolicy(awscdk.RemovalPolicy_DESTROY)

	marioSecret := awssecretsmanager.Secret_FromSecretCompleteArn(
		stack,
		jsii.String("MarioSecret"),
		jsii.String("arn:aws:secretsmanager:ap-south-1:566275025856:secret:mario/defaultSecret-NI1lQX"),
	)
	marioSecret.GrantRead(loginLambda, jsii.Strings())

	marioAuthLogTable := awsdynamodb.NewTable(stack, jsii.String("MarioAuthLog"), &awsdynamodb.TableProps{
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
		Runtime: awslambda.Runtime_PROVIDED_AL2023(),
		Code:    awslambda.AssetCode_FromAsset(jsii.String("lambda/users/function.zip"), nil),
		Handler: jsii.String("main"),
	})
	marioAuthLogTrigger.ApplyRemovalPolicy(awscdk.RemovalPolicy_DESTROY)

	// marioAuth := awsapigatewayv2authorizers.NewHttpUserPoolAuthorizer(
	// 	jsii.String("MarioAuth"),
	// 	awscognito.UserPool_FromUserPoolId(stack, jsii.String("MarioAuth"), jsii.String("ap-south-1_BoauKNKc9")),
	// 	nil,
	// )

	loginAPI := awsapigatewayv2.NewHttpApi(stack, jsii.String("LoginAPI"), &awsapigatewayv2.HttpApiProps{
		ApiName: jsii.String("LambdaLoginAPI"),
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
	loginIntegration := awsapigatewayv2integrations.NewHttpLambdaIntegration(
		jsii.String("LoginIntegration"),
		loginLambda,
		&awsapigatewayv2integrations.HttpLambdaIntegrationProps{},
	)
	loginAPI.AddRoutes(&awsapigatewayv2.AddRoutesOptions{
		Path:        jsii.String("/secret/{id}"),
		Methods:     &[]awsapigatewayv2.HttpMethod{awsapigatewayv2.HttpMethod_GET},
		Integration: loginIntegration,
	})
	loginAPI.ApplyRemovalPolicy(awscdk.RemovalPolicy_DESTROY)

	awscdk.NewCfnOutput(stack, jsii.String("loginAPI URL"), &awscdk.CfnOutputProps{
		Value:       loginAPI.Url(),
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
			Env: env(),
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
