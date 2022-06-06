.PHONY: build

build:
	sam build

validate:
	sam validate --profile adam

sfn-local:
	docker compose \
	-f docker-compose.local.yml \
	up \
	-d

sfn-local-stop:
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

test-happy:
	aws stepfunctions start-execution \
	--endpoint http://localhost:8083 \
	--name "HappyPathExecution" \
	--state-machine "arn:aws:states:us-east-2:123456789012:stateMachine:LocalTesting#HappyPathTest" \
	--input file://local/test-products.json \
	--no-cli-pager

test-empty:
	aws stepfunctions start-execution \
	--endpoint http://localhost:8083 \
	--name "NoStoresExecution" \
	--state-machine "arn:aws:states:us-east-2:123456789012:stateMachine:LocalTesting#NoStoresTest" \
	--input file://local/test-products.json \
	--no-cli-pager

test-db-error:
	aws stepfunctions start-execution \
	--endpoint http://localhost:8083 \
	--name "DBErrorExecution" \
	--state-machine "arn:aws:states:us-east-2:123456789012:stateMachine:LocalTesting#DynamoErrorTest" \
	--input file://local/test-products.json \
	--no-cli-pager

test-all: create test-happy test-empty test-db-error

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

product-checker: build
	sam local invoke \
	--debug \
	--region us-east-2 \
	--event ./local/event-product.json \
	ProductCheckerFunction

message-formatter: build
	sam local invoke \
	--debug \
	--region us-east-2 \
	--event ./local/message-input.json \
	MessageFormatterFunction

message-formatter-empty: build
	sam local invoke \
	--debug \
	--region us-east-2 \
	--event ./local/message-input-empty.json \
	MessageFormatterFunction

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
