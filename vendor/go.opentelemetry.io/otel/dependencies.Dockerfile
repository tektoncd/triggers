# This is a renovate-friendly source of Docker images.
FROM python:3.13.6-slim-bullseye@sha256:e98b521460ee75bca92175c16247bdf7275637a8faaeb2bcfa19d879ae5c4b9a AS python
<<<<<<< HEAD
FROM otel/weaver:v0.20.0@sha256:fa4f1c6954ecea78ab1a4e865bd6f5b4aaba80c1896f9f4a11e2c361d04e197e AS weaver
=======
FROM otel/weaver:v0.17.1@sha256:32523b5e44fb44418786347e9f7dde187d8797adb6d57a2ee99c245346c3cdfe AS weaver
>>>>>>> 77e58354 (vendor deps for opencensus to opentelemetry migration)
FROM avtodev/markdown-lint:v1@sha256:6aeedc2f49138ce7a1cd0adffc1b1c0321b841dc2102408967d9301c031949ee AS markdown
