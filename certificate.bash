#!/bin/bash

#self signed certificate generation for localhost development

echo "Generating self-signed certificate..."
openssl genrsa -out cert.key 2048
openssl ecparam -genkey -name secp384r1 -out cert.key
echo "Creating certificate signing request (CSR)..."
openssl req -new -x509 -sha256 -key cert.key -out cert.crt  -batch -days 365
