cd src || echo "failed cd";exit
go build .
echo "build done"
rm -rf ../run
mv src ../run
cd ..
echo "bin file moved"
