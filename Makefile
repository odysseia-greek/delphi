ROOT_DIR=$(shell pwd)

generate_swagger:
	docker run -it -v ${ROOT_DIR}:${ROOT_DIR} -e SWAGGER_GENERATE_EXTENSION=true --workdir ${ROOT_DIR}/solon quay.io/goswagger/swagger generate spec -o ./docs/swagger.json -m;
	curl -X 'POST' \
      'https://converter.swagger.io/api/convert' \
	  -H 'accept: application/yaml' \
	  -H 'Content-Type: application/json' \
      -d '@./solon/docs/swagger.json' > ./solon/docs/openapi.yaml