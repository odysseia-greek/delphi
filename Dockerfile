FROM alpine

ARG project_name
ARG TARGETOS
ARG TARGETARCH

ENV project_name=${project_name}

RUN apk update && apk add ca-certificates
RUN rm -rf /var/cache/apk/*

RUN mkdir /app
WORKDIR /

COPY dist/bin/${TARGETOS}/${TARGETARCH}/${project_name} /app/${project_name}

ENV TMPDIR=/tmp
ENV GOMAXPROCS=8

ENTRYPOINT [ "sh", "-c", "/app/${project_name}" ]