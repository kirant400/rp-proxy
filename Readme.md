# Proxy for BioTime application

## Run

```
docker build -t rp-proxy:v1.0 .
docker save -o rp-proxy_image.tar rp-proxy:v1.0
docker run --rm -p 8080:8080 --env-file ./.env --name rp-proxy rp-proxy:v1.0
```

- Format code
```
make fmt
```

- Run tests
```
make test
```

- Build binary
```
make build
```

- Run proxy with config
```
make run
```

- Clean binary
```
make clean
```
- Build Docker image
```
make docker-build
```

- Clean dangling Docker images
```
make docker-clean
```

- Run dev environment (default from .env):
```
docker-compose up --build
```

- Run staging:
```
ENVIRONMENT=staging docker-compose up --build
```

- Run prod:
```
ENVIRONMENT=prod docker-compose up --build -d
```

- Stop:
```
docker-compose down
```

- Run in dev (default)
```
make run-dev
```

- Run in staging
```
make run-staging
```

- Run in prod (detached mode)
```
make run-prod
```

- Stop running containers
```
make docker-stop
```

- On PowerShell, if you want to override variables inline, you can still do:
```
make run-staging ENVIRONMENT=staging
```

-On cmd.exe, you may need:
```
set ENVIRONMENT=staging && make run-staging
```
## ✅ Running tests:
```
go test -v
```


