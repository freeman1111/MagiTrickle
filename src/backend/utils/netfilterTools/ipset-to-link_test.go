//go:build testing

package netfilterTools

import (
	"reflect"
	"testing"

	"magitrickle/utils/iptables"

	"github.com/vishvananda/netlink"
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

// TestIPSetToLinkBypassMarks проверяет, что трафик с марками политик не маркируется
func TestIPSetToLinkBypassMarks(t *testing.T) {
	nh := &Helper{ChainPrefix: "MT_", IpsetPrefix: "mt_", Links: []string{"br0"}}
	nh.BypassMarks.Store([]uint32{0xffffaa2, 0xffffaa5})
	r := nh.IPSetToLink("test", "nwg0", nh.IPSet("test"))
	r.mark = 1

	fake := iptables.NewFakeIPTables(iptables.ProtocolIPv4)
	ipt := newTestIPTables(fake)

	if err := r.insertIPTablesRules(ipt); err != nil {
		t.Fatalf("insertIPTablesRules failed: %v", err)
	}

	expected := [][]string{
		{"-m", "mark", "--mark", "0xffffaa2", "-j", "RETURN"},
		{"-m", "mark", "--mark", "0xffffaa5", "-j", "RETURN"},
		{"-m", "conntrack", "--ctdir", "REPLY", "-j", "RETURN"},
		{"-m", "set", "--match-set", "mt_test_4", "dst", "-j", "MARK", "--set-mark", "1"},
		{"-m", "set", "--match-set", "mt_test_4", "dst", "-j", "CONNMARK", "--save-mark"},
	}
	if rules := fake.GetRules("mangle", "MT_test"); !reflect.DeepEqual(rules, expected) {
		t.Errorf("chain rules mismatch.\nExpected: %v\nGot: %v", expected, rules)
	}
}

// TestIPSetToLinkRefreshIPTablesRules проверяет пересборку правил включённой группы после смены меток политик
func TestIPSetToLinkRefreshIPTablesRules(t *testing.T) {
	fake := iptables.NewFakeIPTables(iptables.ProtocolIPv4)
	nh := &Helper{ChainPrefix: "MT_", IpsetPrefix: "mt_", Links: []string{"br0"}, IPTables4: newTestIPTables(fake)}
	nh.BypassMarks.Store([]uint32{0xffffaa2})
	r := nh.IPSetToLink("test", "nwg0", nh.IPSet("test"))
	r.mark = 1

	if err := r.RefreshIPTablesRules(); err != nil {
		t.Fatalf("RefreshIPTablesRules on disabled link failed: %v", err)
	}
	if rules := fake.GetRules("mangle", "MT_test"); rules != nil {
		t.Fatalf("disabled link must not create rules, got: %v", rules)
	}

	r.enabled.Store(true)
	if err := r.insertIPTablesRules(nh.IPTables4); err != nil {
		t.Fatalf("insertIPTablesRules failed: %v", err)
	}

	nh.BypassMarks.Store([]uint32{0xffffaa7})
	if err := r.RefreshIPTablesRules(); err != nil {
		t.Fatalf("RefreshIPTablesRules failed: %v", err)
	}

	expected := [][]string{
		{"-m", "mark", "--mark", "0xffffaa7", "-j", "RETURN"},
		{"-m", "conntrack", "--ctdir", "REPLY", "-j", "RETURN"},
		{"-m", "set", "--match-set", "mt_test_4", "dst", "-j", "MARK", "--set-mark", "1"},
		{"-m", "set", "--match-set", "mt_test_4", "dst", "-j", "CONNMARK", "--save-mark"},
	}
	if rules := fake.GetRules("mangle", "MT_test"); !reflect.DeepEqual(rules, expected) {
		t.Errorf("chain rules mismatch.\nExpected: %v\nGot: %v", expected, rules)
	}

	expectedJumps := [][]string{{"-i", "br0", "-j", "MT_test"}}
	if rules := fake.GetRules("mangle", "PREROUTING"); !reflect.DeepEqual(rules, expectedJumps) {
		t.Errorf("PREROUTING rules must not be duplicated.\nExpected: %v\nGot: %v", expectedJumps, rules)
	}
}

// TestIPSetToLinkPolicyTarget проверяет группу, направленную в политику доступа роутера
func TestIPSetToLinkPolicyTarget(t *testing.T) {
	fake := iptables.NewFakeIPTables(iptables.ProtocolIPv4)
	nh := &Helper{ChainPrefix: "MT_", IpsetPrefix: "mt_", Links: []string{"br0"}, IPTables4: newTestIPTables(fake)}
	nh.Policies.Store(map[string]uint32{"Policy3": 0xffffaae})
	r := nh.IPSetToLink("test", "Policy3", nh.IPSet("test"))

	if err := r.Enable(); err != nil {
		t.Fatalf("Enable failed: %v", err)
	}
	if !r.policy || r.mark != 0xffffaae || r.table != 0 {
		t.Fatalf("policy=%v mark=%#x table=%d, want policy mark 0xffffaae without own table", r.policy, r.mark, r.table)
	}
	if r.ip4Rule != nil || r.ip4Route[0] != nil || r.ip4Route[1] != nil {
		t.Fatalf("policy target must not create own ip rules and routes")
	}
	if rules := fake.GetRules("filter", "MT_test"); len(rules) != 0 {
		t.Errorf("policy target must not add -o rules, got: %v", rules)
	}

	expected := [][]string{
		{"-m", "conntrack", "--ctdir", "REPLY", "-j", "RETURN"},
		{"-m", "set", "--match-set", "mt_test_4", "dst", "-j", "MARK", "--set-mark", "268434094"},
		{"-m", "set", "--match-set", "mt_test_4", "dst", "-j", "CONNMARK", "--save-mark"},
	}
	if rules := fake.GetRules("mangle", "MT_test"); !reflect.DeepEqual(rules, expected) {
		t.Errorf("chain rules mismatch.\nExpected: %v\nGot: %v", expected, rules)
	}

	if err := r.AddrChangeHook(netlink.AddrUpdate{}); err != nil {
		t.Errorf("AddrChangeHook must ignore policy targets, got: %v", err)
	}
	if err := r.LinkUpHook(netlink.LinkUpdate{Link: &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: "Policy3"}}}); err != nil {
		t.Errorf("LinkUpHook must ignore policy targets, got: %v", err)
	}

	nh.Policies.Store(map[string]uint32{"Policy3": 0xffffaa1})
	if err := r.RefreshIPTablesRules(); err != nil {
		t.Fatalf("RefreshIPTablesRules failed: %v", err)
	}
	expected[1] = []string{"-m", "set", "--match-set", "mt_test_4", "dst", "-j", "MARK", "--set-mark", "268434081"}
	if rules := fake.GetRules("mangle", "MT_test"); !reflect.DeepEqual(rules, expected) {
		t.Errorf("chain rules after mark change mismatch.\nExpected: %v\nGot: %v", expected, rules)
	}
	expectedJumps := [][]string{{"-i", "br0", "-j", "MT_test"}}
	if rules := fake.GetRules("mangle", "PREROUTING"); !reflect.DeepEqual(rules, expectedJumps) {
		t.Errorf("PREROUTING rules mismatch.\nExpected: %v\nGot: %v", expectedJumps, rules)
	}

	if err := r.Disable(); err != nil {
		t.Fatalf("Disable failed: %v", err)
	}
	if fake.ChainExists("mangle", "MT_test") || len(fake.GetRules("mangle", "PREROUTING")) != 0 {
		t.Errorf("Disable must remove chain and jumps")
	}
}

// TestIPSetToLinkLongTargetName проверяет, что длинное имя цели (политика до ответа RCI) не ломает правила iptables
func TestIPSetToLinkLongTargetName(t *testing.T) {
	fake := iptables.NewFakeIPTables(iptables.ProtocolIPv4)
	nh := &Helper{ChainPrefix: "MT_", IpsetPrefix: "mt_", Links: []string{"br0"}}
	r := nh.IPSetToLink("test", "VeryLongPolicyName", nh.IPSet("test"))
	r.mark = 1

	if r.routesViaLink() {
		t.Fatalf("name longer than IFNAMSIZ must not be treated as an interface")
	}
	if err := r.insertIPTablesRules(newTestIPTables(fake)); err != nil {
		t.Fatalf("insertIPTablesRules failed: %v", err)
	}
	if rules := fake.GetRules("filter", "MT_test"); len(rules) != 0 {
		t.Errorf("no -o rule expected for invalid interface name, got: %v", rules)
	}
	if err := r.AddrChangeHook(netlink.AddrUpdate{}); err != nil {
		t.Errorf("AddrChangeHook must ignore non-interface targets, got: %v", err)
	}
}
