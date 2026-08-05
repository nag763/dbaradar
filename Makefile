.PHONY: build test clean dev-dba

build:
	$(MAKE) -C services/dba build
	$(MAKE) -C services/analyzer build
	$(MAKE) -C services/telegram-notifier build
	$(MAKE) -C services/webhooks build
	$(MAKE) -C services/authorizer build

test:
	$(MAKE) -C services/dba test
	cd services/analyzer && go test ./...
	cd services/telegram-notifier && go test ./...
	cd services/webhooks && go test -v -race ./...
	$(MAKE) -C services/authorizer test

clean:
	$(MAKE) -C services/dba clean
	$(MAKE) -C services/analyzer clean
	$(MAKE) -C services/telegram-notifier clean
	$(MAKE) -C services/webhooks clean
	$(MAKE) -C services/authorizer clean
