.PHONY: build local

build:
	sam build

validate:
	sam validate --profile adam

go-test:
	go test \
	./functions/historicalStats \
	./functions/productChecker \
	./functions/messageFormatter

go-test-v:
	go test -v \
	./functions/historicalStats \
	./functions/productChecker \
	./functions/messageFormatter

go-vet:
	go vet \
	./functions/historicalStats \
	./functions/productChecker \
	./functions/messageFormatter

local:
	docker compose \
	-f docker-compose.local.yml \
	up \
	-d

local-stop:
	docker compose \
	-f docker-compose.local.yml \
	down

create:
	aws stepfunctions create-state-machine \
	--endpoint http://localhost:8083 \
	--definition file://statemachine/target-tracker.asl.json \
	--name "LocalTesting" \
	--role-arn "arn:aws:iam::123456789012:role/DummyRole" \
	--no-cli-pager

s3-mb:
	aws s3 mb s3://test-historical-bucket \
	--endpoint http://localhost:4566 \
	--profile default

s3-ls:
	aws s3 ls \
	--recursive \
	--endpoint http://localhost:4566 \
	--profile default

s3-cp:
	aws s3 cp s3://test-historical-bucket/historical_stats.json /dev/stdout \
	--quiet \
	--endpoint http://localhost:4566 \
	--profile default \
	| jq . \
	| less

s3-rm:
	aws s3 rm s3://test-historical-bucket/historical_stats.json \
	--endpoint http://localhost:4566 \
	--profile default

test-happy:
	aws stepfunctions start-execution \
	--endpoint http://localhost:8083 \
	--name "HappyPathExecution" \
	--state-machine "arn:aws:states:us-east-2:123456789012:stateMachine:LocalTesting#HappyPathTest" \
	--input file://local/sfn/test-products.json \
	--no-cli-pager

test-empty:
	aws stepfunctions start-execution \
	--endpoint http://localhost:8083 \
	--name "NoStoresExecution" \
	--state-machine "arn:aws:states:us-east-2:123456789012:stateMachine:LocalTesting#NoStoresTest" \
	--input file://local/sfn/test-products.json \
	--no-cli-pager

test-db-error:
	aws stepfunctions start-execution \
	--endpoint http://localhost:8083 \
	--name "DBErrorExecution" \
	--state-machine "arn:aws:states:us-east-2:123456789012:stateMachine:LocalTesting#DynamoErrorTest" \
	--input file://local/sfn/test-products.json \
	--no-cli-pager

test-no-pickup-alert:
	aws stepfunctions start-execution \
	--endpoint http://localhost:8083 \
	--name "NoPickupAlert" \
	--state-machine "arn:aws:states:us-east-2:123456789012:stateMachine:LocalTesting#HappyPathTest" \
	--input file://local/sfn/test-products-no-pickup.json \
	--no-cli-pager

test-all: create test-happy test-empty test-db-error test-no-pickup-alert

hist-happy:
	aws stepfunctions get-execution-history \
	--endpoint http://localhost:8083 \
	--execution-arn "arn:aws:states:us-east-2:123456789012:execution:LocalTesting:HappyPathExecution" \
	--query 'events[?(type==`TaskStateEntered` || type==`TaskStateExited` || type==`MapStateEntered` || type==`MapStateExited` || type==`ExecutionSucceeded` || type==`ExecutionFailed`) || (type==`TaskScheduled` && taskScheduledEventDetails.resourceType==`dynamodb`)]' \
	--no-cli-pager \
	| jq

hist-empty:
	aws stepfunctions get-execution-history \
	--endpoint http://localhost:8083 \
	--execution-arn "arn:aws:states:us-east-2:123456789012:execution:LocalTesting:NoStoresExecution" \
	--query 'events[?(type==`TaskStateEntered` || type==`TaskStateExited` || type==`MapStateEntered` || type==`MapStateExited` || type==`ExecutionSucceeded` || type==`ExecutionFailed`) || (type==`TaskScheduled` && taskScheduledEventDetails.resourceType==`dynamodb`)]' \
	--no-cli-pager \
	| jq

hist-db-error:
	aws stepfunctions get-execution-history \
	--endpoint http://localhost:8083 \
	--execution-arn "arn:aws:states:us-east-2:123456789012:execution:LocalTesting:DBErrorExecution" \
	--query 'events[?(type==`TaskStateEntered` || type==`TaskStateExited` || type==`MapStateEntered` || type==`MapStateExited` || type==`ExecutionSucceeded` || type==`ExecutionFailed`) || (type==`TaskScheduled` && taskScheduledEventDetails.resourceType==`dynamodb`)]' \
	--no-cli-pager \
	| jq
	| jq

hist-no-pickup-alert:
	aws stepfunctions get-execution-history \
	--endpoint http://localhost:8083 \
	--execution-arn "arn:aws:states:us-east-2:123456789012:execution:LocalTesting:NoPickupAlert" \
	--query 'events[?(type==`TaskStateEntered` || type==`TaskStateExited` || type==`MapStateEntered` || type==`MapStateExited` || type==`ExecutionSucceeded` || type==`ExecutionFailed`) || (type==`TaskScheduled` && taskScheduledEventDetails.resourceType==`dynamodb`)]' \
	--no-cli-pager \
	| jq

product-checker: build
	sam local invoke \
	--debug \
	--region us-east-2 \
	--event ./local/lambda/event-product.json \
	ProductCheckerFunction

message-formatter: build
	sam local invoke \
	--debug \
	--region us-east-2 \
	--event ./local/lambda/message-input.json \
	MessageFormatterFunction

message-formatter-empty: build
	sam local invoke \
	--debug \
	--region us-east-2 \
	--event ./local/lambda/message-input-empty.json \
	MessageFormatterFunction

historical-stats: build
	sam local invoke \
	--debug \
	--region us-east-2 \
	--event ./local/lambda/message-input.json \
	--env-vars ./local/lambda/stats-env.json \
	--docker-network aws-sam-local \
	--region us-east-2 \
	HistoricalStatsFunction

deploy-stage: build
	sam deploy \
	--config-env stage

deploy-stage-dry: build
	sam deploy \
	--no-execute-changeset \
	--config-env stage

delete-stage:
	sam delete \
	--config-env stage
