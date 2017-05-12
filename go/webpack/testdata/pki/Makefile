OPENSSL_CONF:=openssl.cnf
REQ:=openssl req -config $(OPENSSL_CONF)
CA:=openssl ca -config $(OPENSSL_CONF) -batch

all: example.com.cert

IMPORTANT_FILES:=Makefile openssl.cnf
clean:
	rm -f $(filter-out $(IMPORTANT_FILES), $(wildcard *))

%.password:
	openssl rand -out $@ -base64 16

database:
	echo -n > $@

serial:
	echo 01 > $@

root1.key: root1.password
	openssl genrsa -out root1.key -passout file:root1.password 4096

# The "|" makes sure the database and serial files exist, but doesn't rebuild
# when they change.
root1.cert: root1.key | database serial
	$(REQ) -new -subj "/C=US/O=Honest Achmed's/CN=Honest Achmed's Test Root CA" -key root1.key -passin file:root1.password -out root1.certreq
	@# We avoid -x509 above so that the startdate gets set to the value in openssl.cnf.
	$(CA) -selfsign -extensions v3_root_ca -keyfile root1.key -passin file:root1.password -in root1.certreq -out root1.cert

intermediate1.key: intermediate1.password
	openssl ecparam -genkey -name secp521r1 -noout | openssl ec -aes-128-cbc -passout file:intermediate1.password -out intermediate1.key

intermediate1.cert: intermediate1.key root1.cert
	$(REQ) -new -subj "/C=US/O=Honest Achmed's/CN=Honest Achmed's Test Intermediate CA" -key intermediate1.key -passin file:intermediate1.password -out intermediate1.certreq
	$(CA) -cert root1.cert -keyfile root1.key -extensions v3_intermediate_ca -passin file:root1.password -in intermediate1.certreq -out intermediate1.cert

example.com.key: example.com.password
	openssl ecparam -genkey -name secp256r1 -noout | openssl ec -aes-128-cbc -passout file:example.com.password -out example.com.key

example.com.cert: example.com.key intermediate1.cert
	$(REQ) -new -subj "/C=US/O=Test Example/CN=example.com" -key example.com.key -passin file:example.com.password -out example.com.certreq
	$(CA) -cert intermediate1.cert -keyfile intermediate1.key -extensions server_cert -passin file:intermediate1.password -in example.com.certreq -out example.com.cert

check: all
	openssl verify -attime $(shell date --date="2017-05-17 00:00:00 GMT" +%s) -CAfile root1.cert -untrusted intermediate1.cert -purpose sslserver example.com.cert


.DELETE_ON_ERROR:
.SUFFIXES:
