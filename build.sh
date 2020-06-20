cd src || echo "failed cd";exit
go build .
rm -rf ../run
mv src ../run
cd ..
