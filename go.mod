module github.com/getcasa/plugin-yeelight

go 1.12

require (
	github.com/anvie/port-scanner v0.0.0-20180225151059-8159197d3770
	github.com/bcurren/go-ssdp v0.0.0-20130927201504-ae8e7a0ef8a8
	github.com/getcasa/sdk v0.0.0-20191122192853-83858676b651
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/koron/go-ssdp v0.0.0-20191105050749-2e1c40ed0b5d
	github.com/labstack/gommon v0.3.0
	github.com/liamg/furious v0.0.0-20190619180719-b76d3ae59fcc // indirect
	github.com/pkg/errors v0.8.1
	github.com/pulento/go-ssdp v0.0.0-20180514024734-4a0ed625a78b
	github.com/pulento/yeelight v0.0.0-20180827013714-e72aa2e3c4ef
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/net v0.0.0-20190923162816-aa69164e4478
	golang.org/x/sys v0.0.0-20190924154521-2837fb4f24fe // indirect
)

replace github.com/getcasa/sdk v0.0.0-20191105095754-6df142bc28a9 => ../casa-sdk
