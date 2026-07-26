.PHONY: build test clean dev-dba

build:
	$(MAKE) -C services/dba build-native
	$(MAKE) -C services/analyzer build
	$(MAKE) -C services/telegram-notifier build

test:
	$(MAKE) -C services/dba test
	cd services/analyzer && go test ./...
	cd services/telegram-notifier && go test ./...

clean:
	$(MAKE) -C services/dba clean
	$(MAKE) -C services/analyzer clean
	$(MAKE) -C services/telegram-notifier clean

dev-dba:
	$(MAKE) -C services/dba dev