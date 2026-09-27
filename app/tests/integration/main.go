package main

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// https://medium.com/@dilshataliev/integration-tests-with-golang-test-containers-and-postgres-abb49e8096c5
	os.Exit(m.Run())
}

func TestCreateOrder(t *testing.T) {

}

func TestGetOrders(t *testing.T) {

}
