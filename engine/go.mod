module github.com/ds3lab/easeml/engine

go 1.14

require (
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Microsoft/go-winio v0.4.12 // indirect
	github.com/bmizerany/pat v0.0.0-20170815010413-6226ea591a40 // indirect
	github.com/cavaliercoder/grab v2.0.0+incompatible
	github.com/containerd/containerd v1.3.2 // indirect
	github.com/coreos/go-etcd v2.0.0+incompatible // indirect
	github.com/cpuguy83/go-md2man v1.0.10 // indirect
	github.com/docker/docker v1.4.2-0.20200103225628-a9507c6f7662
	github.com/ds3lab/easeml/client/go/easemlclient v0.0.0
	github.com/ds3lab/easeml/schema/go/easemlschema v0.0.0
	github.com/emicklei/forest v1.1.0
	github.com/ghodss/yaml v1.0.0
	github.com/globalsign/mgo v0.0.0-20181015135952-eeefdecb41b8
	github.com/gobuffalo/envy v1.9.0 // indirect
	github.com/gobuffalo/packr/v2 v2.8.0
	github.com/golang/mock v1.3.0 // indirect
	github.com/google/go-cmp v0.3.0 // indirect
	github.com/gorilla/context v1.1.1
	github.com/gorilla/mux v1.7.1
	github.com/howeyc/gopass v0.0.0-20170109162249-bf9dde6d0d2c
	github.com/justinas/alice v0.0.0-20171023064455-03f45bd4b7da
	github.com/karrick/godirwalk v1.15.6 // indirect
	github.com/mholt/archiver v2.1.0+incompatible
	github.com/mitchellh/go-homedir v1.1.0
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/otiai10/copy v1.0.1
	github.com/otiai10/curr v0.0.0-20150429015615-9b4961190c95 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.8.1
	github.com/rogpeppe/go-internal v1.6.0 // indirect
	github.com/rs/cors v1.6.0
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.5.1
	github.com/tus/tusd v0.0.0-20190508030626-9d693c93a3ea
	github.com/ugorji/go/codec v0.0.0-20181204163529-d75b2dcb6bc8 // indirect
	golang.org/x/crypto v0.0.0-20200709230013-948cd5f35899 // indirect
	golang.org/x/net v0.0.0-20200625001655-4c5254603344
	golang.org/x/sync v0.0.0-20200625203802-6e8e738ad208 // indirect
	golang.org/x/sys v0.0.0-20200728102440-3e129f6d46b1 // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	golang.org/x/tools v0.0.0-20200728160517-2ad651e9e297 // indirect
	gopkg.in/Acconut/lockfile.v1 v1.1.0 // indirect
	gotest.tools v2.2.0+incompatible // indirect
)

replace github.com/Sirupsen/logrus v1.4.1 => github.com/sirupsen/logrus v1.4.1

replace github.com/Sirupsen/logrus v1.4.2 => github.com/sirupsen/logrus v1.4.2

replace github.com/ds3lab/easeml/client/go/easemlclient v0.0.0 => ../client/go/easemlclient

replace github.com/ds3lab/easeml/schema/go/easemlschema v0.0.0 => ../schema/go/easemlschema
