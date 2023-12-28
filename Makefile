APPNAME := cloudfront-simple-analytics
STAGE ?= dev
BRANCH ?= master

TAG ?= $(shell git rev-parse --short HEAD)

.PHONY: default
default: clean build archive deploy

.PHONY: ci
ci: clean test

LDFLAGS := -ldflags="-s -w -X main.version=${TAG}"

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
	@sam deploy \
		--no-fail-on-empty-changeset \
		--template-file sam/website/api.cfn.yaml \
		--capabilities CAPABILITY_IAM \
		--s3-bucket $(SAM_BUCKET) \
		--s3-prefix sam/$(GIT_HASH) \
		--tags "environment=$(STAGE)" "branch=$(BRANCH)" "service=$(APPNAME)" \
		--stack-name $(APPNAME)-api-$(STAGE)-$(BRANCH) \
		--parameter-overrides AppName=$(APPNAME) Stage=$(STAGE) Branch=$(BRANCH)


.PHONY: api-logs
api-logs:
	@sam logs --stack-name $(APPNAME)-api-$(STAGE)-$(BRANCH) --tail
