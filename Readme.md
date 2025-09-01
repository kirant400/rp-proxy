# Proxy for BioTime application

```
docker build -t rp-proxy:v1.0 .
docker save -o rp-proxy_image.tar rp-proxy:v1.0
docker run --rm -p 8080:8080 --env-file ./.env --name rp-proxy rp-proxy:v1.0
```