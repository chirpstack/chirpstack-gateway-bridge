#!/usr/bin/env bash

echo '/*' > doc.go
go run main.go help >> doc.go
echo >> doc.go
echo '*/' >> doc.go
echo -e "package main" >> doc.go
