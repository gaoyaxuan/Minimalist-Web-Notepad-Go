# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:alpine AS build
WORKDIR /src
COPY . .
ARG TARGETOS TARGETARCH
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH CGO_ENABLED=0 go build -a -o Minimalist-Web-Notepad-Go  ./main.go

FROM --platform=${TARGETPLATFORM} alpine:latest
WORKDIR /app
COPY --from=build /src/Minimalist-Web-Notepad-Go /app/Minimalist-Web-Notepad-Go
COPY --from=build /src/index.html  /app/
COPY --from=build /src/static/*  /app/static/
RUN apk --no-cache --no-progress add tzdata \
    && set -ex \
    && chmod 544 /app/Minimalist-Web-Notepad-Go \
    &&  chmod 444 /app/index.html \
    && chmod -R 444 /app/static/*



ENV TZ=Asia/Shanghai
CMD [ "/app/Minimalist-Web-Notepad-Go" ]
