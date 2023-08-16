# Release New AMI to Auto Scaling Group

### CodePipeline Structure :
- Stage : Source
    - Action :
        - Category : Source
        - Provider : CodeStarSourceConnection
- Stage : Build
    - Action :
        - Category : Build
        - Provider : Codebuild
- Stage : Deploy
    - Action :
        - Category : Invoke
        - Provider : Lambda <=== this repository

### Create Executable With : 
```
env GOOS=linux GOARCH=arm64 go build -o bootstrap
```
notes :
- adjust GOARCH as necessary, e.g. : `amd64`
- executable name must be `bootstrap` if targeting `provided.al2` runtime [[Reference]](https://docs.aws.amazon.com/lambda/latest/dg/golang-package.html)
