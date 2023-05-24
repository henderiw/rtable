package main

import (
	"fmt"
	"net/netip"

	"github.com/hansthienpondt/nipam/pkg/table"
	allocv1alpha1 "github.com/nokia/k8s-ipam/apis/alloc/common/v1alpha1"
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/alloc/ipam/v1alpha1"
	invv1alpha1 "github.com/nokia/k8s-ipam/apis/inv/v1alpha1"
	"github.com/nokia/k8s-ipam/pkg/iputil"
)

func main() {
	prefixes := []ipamv1alpha1.Prefix{
		{
			Prefix: "172.0.0.0/16",
		},
		{
			Prefix: "10.0.0.0/8",
			UserDefinedLabels: allocv1alpha1.UserDefinedLabels{
				Labels: map[string]string{
					allocv1alpha1.NephioPrefixKindKey: string(ipamv1alpha1.PrefixKindPool),
				},
			},
		},
	}
	clusters := []string{"cluster01", "cluster02", "cluster03", "cluster04"}

	rtable := table.NewRIB()
	for _, prefix := range prefixes {
		mpi := iputil.NewPrefixInfo(netip.MustParsePrefix(prefix.Prefix))
		// defaults to prefixKind network
		prefixKind := ipamv1alpha1.PrefixKindNetwork
		prefixLength := 24
		labels := map[string]string{
			allocv1alpha1.NephioPrefixKindKey: string(ipamv1alpha1.PrefixKindNetwork),
		}
		if k, ok := prefix.Labels[allocv1alpha1.NephioPrefixKindKey]; ok {
			labels[allocv1alpha1.NephioPrefixKindKey] = k
			prefixKind = ipamv1alpha1.GetPrefixKindFromString(k)
			prefixLength = 16
		}
		// add additional labels from the prefix Spec
		for k, v := range prefix.Labels {
			labels[k] = v
		}
		// add the route to the routing table
		route := table.NewRoute(mpi.GetIPPrefix(), labels, nil)
		rtable.Add(route)

		for _, clusterName := range clusters {
			p := rtable.GetAvailablePrefixByBitLen(mpi.GetIPPrefix(), uint8(prefixLength))
			// add clusterName to the labels
			labels := getClusterLabels(labels, clusterName)
			
			// the default gw is .1 of the .24
			pi := iputil.NewPrefixInfo(netip.PrefixFrom(p.Addr().Next(), prefixLength))
			if prefixKind != ipamv1alpha1.PrefixKindNetwork {
				// update labels for now gateways
				delete(labels, allocv1alpha1.NephioGatewayKey)
				pi = iputil.NewPrefixInfo(netip.PrefixFrom(p.Addr(), prefixLength))
			}
			route := table.NewRoute(pi.GetIPPrefix(), labels, nil)
			rtable.Add(route)
		}
	}

	for _, route := range rtable.GetTable() {
		fmt.Println(route)
	}
}

func getClusterLabels(labels map[string]string, clusterName string) map[string]string {
	l := map[string]string{}
	for k, v := range labels {
		l[k] = v
	}
	l[invv1alpha1.NephioClusterNameKey] = clusterName
	// update labels - defaulting to prefixkind = network and gateway true
	l[allocv1alpha1.NephioGatewayKey] = "true"
	return l
}
