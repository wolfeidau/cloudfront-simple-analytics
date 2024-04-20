APPNAME := cloudfront-simple-analytics
STAGE ?= dev
BRANCH ?= master

GIT_HASH := $(shell git rev-parse --short HEAD)

.PHONY: default
default: clean build archive deploy

.PHONY: ci
ci: clean test

LDFLAGS := -ldflags="-s -w -X main.version=${GIT_HASH}"

.PHONY: clean
clean:
	@echo "--- clean all the things"
	@rm -rf ./dist

.PHONY: test
test:
	@echo "--- test all the things"
	@go test -coverprofile=coverage.txt ./...
	@go tool cover -func=coverage.txt

.PHONY: build
build:
	@echo "--- build all the things"
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -tags lambda.norpc -trimpath -o build/ ./cmd/...

.PHONY: archive
archive:
	@echo "--- build an archive"
	@mkdir -p ./dist
	@go run github.com/wolfeidau/lambdapack ./build ./dist

.PHONY: deploy-api
deploy-api: clean build archive
	@echo "--- deploy stack $(APPNAME)-api-$(STAGE)-$(BRANCH)"
	$(eval SAM_BUCKET := $(shell aws ssm get-parameter --name '/config/dev/master/deploy_bucket' --query 'Parameter.Value' --output text))
	$(eval FIREHOSE_IMPORT_STREAM_NAME := $(shell aws ssm get-parameter --name '/config/${STAGE}/${BRANCH}/${APPNAME}/analytics_events_firehose_import_stream_name' --query 'Parameter.Value' --output text))
	@sam deploy \
		--no-fail-on-empty-changeset \
		--template-file sam/website/api.cfn.yaml \
		--capabilities CAPABILITY_IAM \
		--s3-bucket $(SAM_BUCKET) \
		--s3-prefix sam/$(GIT_HASH) \
		--tags "environment=$(STAGE)" "branch=$(BRANCH)" "service=$(APPNAME)" \
		--stack-name $(APPNAME)-api-$(STAGE)-$(BRANCH) \
		--parameter-overrides AppName=$(APPNAME) Stage=$(STAGE) Branch=$(BRANCH) \
			"FirehoseImportStreamName=$(FIREHOSE_IMPORT_STREAM_NAME)"

.PHONY: deploy-cloudfront-acm
deploy-cloudfront-acm:
	@sam deploy \
		--region us-east-1 \
		--no-fail-on-empty-changeset \
		--template-file sam/website/acm.cfn.yaml \
		--capabilities CAPABILITY_IAM \
		--stack-name $(APPNAME)-$(STAGE)-$(BRANCH)-acm-certificate \
		--tags "environment=$(STAGE)" "branch=$(BRANCH)" "application=$(APPNAME)" \
		--parameter-overrides \
			"AppName=$(APPNAME)" \
			"Stage=$(STAGE)" \
			"Branch=$(BRANCH)" \
			"PrimarySubDomainName=$(PRIMARY_SUB_DOMAIN_NAME)" \
			"HostedZoneName=$(HOSTED_ZONE_NAME)" \
			"HostedZoneId=$(HOSTED_ZONE_ID)" \
			"CertificateName=cloudfront"

.PHONY: deploy-cloudfront-website
deploy-cloudfront-website:
	$(eval ACM_CERTIFICATE_ARN := $(shell aws --region us-east-1 ssm get-parameter --name '/config/${STAGE}/${BRANCH}/${APPNAME}/cloudfront/acm_certificate' --query 'Parameter.Value' --output text))
	$(eval ANALYTICS_FUNCTION_URL := $(shell aws ssm get-parameter --name '/config/$(STAGE)/$(BRANCH)/$(APPNAME)/analytics_api_function_url' --query 'Parameter.Value' --output text | cut -d'/' -f3 | cut -d':' -f1))
	$(eval ANALYTICS_FUNCTION_ARN := $(shell aws ssm get-parameter --name '/config/$(STAGE)/$(BRANCH)/$(APPNAME)/analytics_api_function_arn' --query 'Parameter.Value' --output text))
	@sam deploy \
		--no-fail-on-empty-changeset \
		--template-file sam/website/cloudfront-static-website.cfn.yaml \
		--capabilities CAPABILITY_IAM \
		--stack-name $(APPNAME)-$(STAGE)-$(BRANCH)-static-website \
		--tags "environment=$(STAGE)" "branch=$(BRANCH)" "application=$(APPNAME)" \
		--parameter-overrides \
			"AppName=$(APPNAME)" "Stage=$(STAGE)" "Branch=$(BRANCH)" \
			"AnalyticsFunctionUrl=$(ANALYTICS_FUNCTION_URL)" \
			"AnalyticsFunctionArn=$(ANALYTICS_FUNCTION_ARN)" \
			"PrimarySubDomainName=$(PRIMARY_SUB_DOMAIN_NAME)" \
			"HostedZoneName=$(HOSTED_ZONE_NAME)" \
			"HostedZoneId=$(HOSTED_ZONE_ID)" \
			"AcmCertificateArn=$(ACM_CERTIFICATE_ARN)"

.PHONY: deploy-analytics-database
deploy-analytics-database:
	@sam deploy \
		--no-fail-on-empty-changeset \
		--template-file sam/datalake/analytics_athena_database.cfn.yaml \
		--capabilities CAPABILITY_IAM \
		--stack-name $(APPNAME)-$(STAGE)-$(BRANCH)-analytics-database \
		--tags "environment=$(STAGE)" "branch=$(BRANCH)" "application=$(APPNAME)" \
		--parameter-overrides \
			"AppName=$(APPNAME)" \
			"Stage=$(STAGE)" \
			"Branch=$(BRANCH)" \
			"DatabaseName=bare_website_analytics"

.PHONY: deploy-analytics-import-table
deploy-analytics-import-table:
	$(eval ANALYTICS_DATABASE_NAME := $(shell aws ssm get-parameter --name '/config/$(STAGE)/$(BRANCH)/$(APPNAME)/analytics_glue_database' --query 'Parameter.Value' --output text))
	@sam deploy \
		--no-fail-on-empty-changeset \
		--template-file sam/datalake/analytics_athena_import_table.cfn.yaml \
		--capabilities CAPABILITY_IAM \
		--stack-name $(APPNAME)-$(STAGE)-$(BRANCH)-analytics-import-table \
		--tags "environment=$(STAGE)" "branch=$(BRANCH)" "application=$(APPNAME)" \
		--parameter-overrides \
			"AppName=$(APPNAME)" \
			"Stage=$(STAGE)" \
			"Branch=$(BRANCH)" \
			"AnalyticsDatabaseName=$(ANALYTICS_DATABASE_NAME)"

.PHONY: deploy-analytics-event-website-stream
deploy-analytics-event-website-stream: clean build archive
	$(eval SAM_BUCKET := $(shell aws ssm get-parameter --name '/config/dev/master/deploy_bucket' --query 'Parameter.Value' --output text))
	$(eval ANALYTICS_DATABASE_NAME := $(shell aws ssm get-parameter --name '/config/$(STAGE)/$(BRANCH)/$(APPNAME)/analytics_glue_database' --query 'Parameter.Value' --output text))
	$(eval ANALYTICS_EVENTS_TABLE_NAME := $(shell aws ssm get-parameter --name '/config/$(STAGE)/$(BRANCH)/$(APPNAME)/analytics_events_import_glue_table' --query 'Parameter.Value' --output text))
	$(eval ANALYTICS_EVENTS_IMPORT_BUCKET_NAME := $(shell aws ssm get-parameter --name '/config/$(STAGE)/$(BRANCH)/$(APPNAME)/analytics_logs_import_bucket_name' --query 'Parameter.Value' --output text))
	@sam deploy \
		--no-fail-on-empty-changeset \
		--template-file sam/datalake/analytics_event_website_stream.cfn.yaml \
		--s3-bucket $(SAM_BUCKET) \
		--s3-prefix sam/$(GIT_HASH) \
		--capabilities CAPABILITY_IAM \
		--stack-name $(APPNAME)-$(STAGE)-$(BRANCH)-analytics-event-website-stream \
		--tags "environment=$(STAGE)" "branch=$(BRANCH)" "application=$(APPNAME)" \
		--parameter-overrides \
			"AppName=$(APPNAME)" \
			"Stage=$(STAGE)" \
			"Branch=$(BRANCH)" \
			"AnalyticsDatabaseName=$(ANALYTICS_DATABASE_NAME)" \
			"AnalyticsEventsImportTableName=$(ANALYTICS_EVENTS_TABLE_NAME)" \
			"AnalyticsEventsImportBucketName=$(ANALYTICS_EVENTS_IMPORT_BUCKET_NAME)" \
			"PrimarySubDomainName=$(PRIMARY_SUB_DOMAIN_NAME)" \
			"HostedZoneName=$(HOSTED_ZONE_NAME)"

.PHONY: api-logs
api-logs:
	@sam logs --stack-name $(APPNAME)-api-$(STAGE)-$(BRANCH) --tail
