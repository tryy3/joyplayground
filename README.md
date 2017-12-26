# Playground for joy
Playground for https://github.com/matthewmueller/joy based on https://github.com/golang/playground and https://github.com/gopherjs/gopherjs.github.io/

## Installation
### Prerequisites 
 * AWS Lambda function
 * S3 bucket
 * Go
 * Docker

### Installation
Start with installing the aws-lambda-go-shim

```
docker pull eawsy/aws-lambda-go-shim:latest
go get -u -d github.com/eawsy/aws-lambda-go-core/...
```

Next compile the go code by running the makefile `make`

Now you can deploy the backend by uploading the handler.zip to your AWS Lambda function.
Make sure the Lambda function is configured like: 
 - Runtime: python2.7
 - Handler: handler.Handle

The lambda function also need 3 environment variables to work:
 - GOPATH: /tmp
 - GOROOT: /tmp
 - S3_BUCKET: your s3 bucket name

Once the backend is up and running you will need to configure an API gateway for your lambda function, so your frontend can talk to your backend.
Configure the API gateway to look something like this:

```
{
    "paths": {
        "/compile": {
            "methods": ["post", "options"]
        },
        "/fmt": {
            "methods": ["post", "options"]
        },
        "/p/{id}": {
            "methods": ["get", "options"]
        },
        "/share": {
            "methods": ["post", "options"]
        },
    },
}
```

Make sure to enable CORS on the paths.

Now that the backend is deployed you can configure the frontend. The frontend is just static files that communicate with the backend through ajax.

So all you need to do is upload the files to a webserver like apache or netlify.com, make sure to edit the API url in playground.js so the ajax requests goes to your lambda function.

## Features
 * Deploy the joy playground on backends such as S3 and lambda
 * Deploy the frontend in most environments such as netlify.com
 * The playground have support for fmt, embed, share, compile
 * Support for multiple themes (right now it only supports dark mode and light mode)

## Future plans
This is a list of things we would like to look into, some might get implemented while others might not.

 - Rewrite the frontend into Go and compile with joy
 - Live compiling
 - Go get support?
 - Add more support for multiple stores, right now it supports reading from s3, local storage and golang's playground but it only supports writing to s3.
 - More test coverage
