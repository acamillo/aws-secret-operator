build:
	go build cmd/...

# Example usage: make docker-build tag=v0.0.1
docker-build: build
	operator-sdk build acamillo/aws-secret-operator:${tag}

docker-login:
	`aws ecr get-login --no-include-email --region us-east-1`

# Example usage: make docker-push tag=v0.0.2
docker-push: docker-build docker-login
    docker push acamillo/aws-secret-operator:${tag}
