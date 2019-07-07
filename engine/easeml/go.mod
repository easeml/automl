module github.com/ds3lab/easeml/engine/easeml

go 1.12.5

require (
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Microsoft/go-winio v0.4.12 // indirect
	github.com/Nvveen/Gotty v0.0.0-20120604004816-cd527374f1e5 // indirect
	github.com/bmizerany/pat v0.0.0-20170815010413-6226ea591a40 // indirect
	github.com/cavaliercoder/grab v2.0.0+incompatible
	github.com/cpuguy83/go-md2man v1.0.10 // indirect
	github.com/docker/docker v0.7.3-0.20190121142752-69d9ff34556d
	github.com/ds3lab/easeml/client/go/easemlclient v0.0.0
	github.com/ds3lab/easeml/schema/go/easemlschema v0.0.0
	github.com/emicklei/forest v1.1.0
	github.com/ghodss/yaml v1.0.0
	github.com/globalsign/mgo v0.0.0-20181015135952-eeefdecb41b8
	github.com/gobuffalo/packr v1.25.0
	github.com/golang/mock v1.3.0 // indirect
	github.com/google/go-cmp v0.3.0 // indirect
	github.com/gorilla/context v1.1.1
	github.com/gorilla/mux v1.7.1
	github.com/howeyc/gopass v0.0.0-20170109162249-bf9dde6d0d2c
	github.com/justinas/alice v0.0.0-20171023064455-03f45bd4b7da
	github.com/mholt/archiver v2.1.0+incompatible
	github.com/mitchellh/go-homedir v1.1.0
	github.com/otiai10/copy v1.0.1
	github.com/otiai10/curr v0.0.0-20150429015615-9b4961190c95 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.8.1
	github.com/rs/cors v1.6.0
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.4.1
	github.com/spf13/cobra v0.0.3
	github.com/spf13/viper v1.3.2
	github.com/stretchr/testify v1.3.0
	github.com/tus/tusd v0.0.0-20190508030626-9d693c93a3ea
	golang.org/x/net v0.0.0-20190503192946-f4e77d36d62c
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	google.golang.org/grpc v1.20.1 // indirect
	gopkg.in/Acconut/lockfile.v1 v1.1.0 // indirect
	gotest.tools v2.2.0+incompatible // indirect
)

replace github.com/Sirupsen/logrus v1.1.1 => github.com/sirupsen/logrus v1.1.1

replace github.com/ds3lab/easeml/client/go/easemlclient v0.0.0 => ../../client/go/easemlclient

replace github.com/ds3lab/easeml/schema/go/easemlschema v0.0.0 => ../../schema/go/easemlschema
