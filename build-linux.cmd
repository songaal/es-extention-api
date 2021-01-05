SET GOOS=linux

del dist\\application

call go build -o dist\\application cmd\\es-extention-api\\main.go

echo finish
