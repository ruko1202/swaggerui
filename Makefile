.PHONY: update-assets

update-assets:
	go run ./generator/generate.go

run_test:
	cd ./example && go run ./...