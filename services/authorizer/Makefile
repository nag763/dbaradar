.PHONY: build clean test

build:
	cargo lambda build -l build --release --arm64
	cd build && zip authorizer.zip authorizer/bootstrap -j

clean:
	rm -rf build/

test:
	cargo test
