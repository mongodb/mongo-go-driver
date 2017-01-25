package core

import "testing"

func TestClusterMonitorFSM(t *testing.T) {

	fsm := &clusterMonitorFSM{}

	uri, _ := ParseURI("mongodb://a")
	for _, host := range uri.Hosts {
		fsm.addServer(Endpoint(host))
	}

}
