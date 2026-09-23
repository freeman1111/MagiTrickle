//go:build testing

package netfilterTools

import (
	"reflect"
	"testing"

	"magitrickle/utils/iptables"
)

func newTestIPTables(fake *iptables.FakeIPTables) *iptables.IPTables {
	ipt := iptables.NewIPTables(fake)
	ipt.RegisterChainPatch("filter", "FORWARD")
	ipt.RegisterChainPatch("mangle", "PREROUTING")
	ipt.RegisterChainPatch("nat", "POSTROUTING")
	return ipt
}

// TestIPSetToLinkRoutesOnlyLinks проверяет, что маркировка подключается только к указанным интерфейсам
func TestIPSetToLinkRoutesOnlyLinks(t *testing.T) {
	nh := &Helper{ChainPrefix: "MT_", IpsetPrefix: "mt_", Links: []string{"br0", "br1"}}
	r := nh.IPSetToLink("test", "nwg0", nh.IPSet("test"))
	r.mark = 1

	fake := iptables.NewFakeIPTables(iptables.ProtocolIPv4)
	ipt := newTestIPTables(fake)

	if err := r.insertIPTablesRules(ipt); err != nil {
		t.Fatalf("insertIPTablesRules failed: %v", err)
	}

	expected := [][]string{
		{"-i", "br0", "-j", "MT_test"},
		{"-i", "br1", "-j", "MT_test"},
	}
	if rules := fake.GetRules("mangle", "PREROUTING"); !reflect.DeepEqual(rules, expected) {
		t.Errorf("PREROUTING rules mismatch.\nExpected: %v\nGot: %v", expected, rules)
	}

	if err := r.deleteIPTablesRules(ipt); err != nil {
		t.Fatalf("deleteIPTablesRules failed: %v", err)
	}

	if rules := fake.GetRules("mangle", "PREROUTING"); len(rules) != 0 {
		t.Errorf("PREROUTING rules should be removed, got: %v", rules)
	}
}
