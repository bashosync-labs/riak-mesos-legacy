// +build rel

//go:generate go-bindata -ignore=Makefile -o bindata_generated.go -pkg=scheduler -prefix=data/ data/

package scheduler
