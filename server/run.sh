# make sure to run:
# $ chmod +x run.sh

rm ./archer-server;
go mod tidy;
go build;
./archer-server;