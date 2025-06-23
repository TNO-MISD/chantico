.PHONY: build push

TAG=ci.tno.nl:4567/ipcei-cis-misd-sustainable-datacenters/wp2/energy-domain-controller/chantico/chantico-aggregator
VERSION=0.0.2

build: chantico.dockerfile
	docker build --tag $(TAG):$(VERSION) -f $< .

push:
	docker push $(TAG):$(VERSION)
