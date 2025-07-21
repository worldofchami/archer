# make sure to run:
# $ chmod +x run.sh

rm ./archer-cli;
go mod tidy;
go build;
./archer-cli;