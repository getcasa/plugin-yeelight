module github.com/getcasa/plugin-xiaomi

go 1.12

require (
	github.com/getcasa/sdk v0.0.0-20190923145410-20bbee062dc8
	github.com/pulento/go-ssdp v0.0.0-20180514024734-4a0ed625a78b // indirect
	github.com/pulento/yeelight v0.0.0-20180827013714-e72aa2e3c4ef
	github.com/sirupsen/logrus v1.4.2 // indirect
	golang.org/x/net v0.0.0-20190923162816-aa69164e4478 // indirect
)

replace github.com/getcasa/sdk v0.0.0-20190923145410-20bbee062dc8 => ../casa-sdk
