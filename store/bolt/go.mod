module github.com/stevecallear/salsa/store/bolt

go 1.18

replace github.com/stevecallear/salsa => ../../

require (
	github.com/google/uuid v1.3.0
	github.com/stevecallear/salsa v0.0.0-00010101000000-000000000000
	go.etcd.io/bbolt v1.3.6
)

require golang.org/x/sys v0.0.0-20200923182605-d9f96fdee20d // indirect
