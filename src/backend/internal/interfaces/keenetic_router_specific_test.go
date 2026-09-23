//go:build entware_kn

package interfaces

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKeeneticRouterSpecificAPIGetIfaceAliasesUsesBatchRCI(t *testing.T) {
	systemNameRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rci/show/interface":
			if err := json.NewEncoder(w).Encode(map[string]keeneticInterfaceMeta{
				"Wireguard0": {Description: "Home VPN"},
				"Wireguard1": {InterfaceName: "Backup VPN"},
			}); err != nil {
				t.Fatalf("encode interface list: %v", err)
			}
		case "/rci/":
			systemNameRequests++
			if r.Method != http.MethodPost {
				t.Fatalf("system-name request method = %s, want POST", r.Method)
			}

			var requests []keeneticInterfaceSystemNameRequest
			if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
				t.Fatalf("decode system-name request: %v", err)
			}
			if len(requests) != 2 {
				t.Fatalf("system-name requests count = %d, want 2", len(requests))
			}

			responseByName := map[string]string{
				"Wireguard0": "nwg0",
				"Wireguard1": "nwg1",
			}
			response := make([]map[string]map[string]map[string]string, len(requests))
			for i, req := range requests {
				response[i] = map[string]map[string]map[string]string{
					"show": {
						"interface": {
							"system-name": responseByName[req.Show.Interface.Name],
						},
					},
				}
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Fatalf("encode system-name response: %v", err)
			}
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	api := &KeeneticRouterSpecificAPI{
		BaseURL: server.URL,
		Client:  server.Client(),
	}
	aliases, err := api.GetIfaceAliases()
	if err != nil {
		t.Fatalf("GetIfaceAliases returned error: %v", err)
	}

	if systemNameRequests != 1 {
		t.Fatalf("system-name request count = %d, want 1", systemNameRequests)
	}
	if aliases["nwg0"] != "Home VPN" {
		t.Fatalf("aliases[nwg0] = %q, want Home VPN", aliases["nwg0"])
	}
	if aliases["nwg1"] != "Backup VPN" {
		t.Fatalf("aliases[nwg1] = %q, want Backup VPN", aliases["nwg1"])
	}
}

func TestKeeneticRouterSpecificAPIGetPolicyMark(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rci/show/ip/policy/noMT/mark":
			if err := json.NewEncoder(w).Encode("ffffaa2"); err != nil {
				t.Fatalf("encode policy mark: %v", err)
			}
		case "/rci/show/ip/policy/Policy0/mark":
			if err := json.NewEncoder(w).Encode("0xffffaa1"); err != nil {
				t.Fatalf("encode policy mark: %v", err)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	api := &KeeneticRouterSpecificAPI{
		BaseURL: server.URL,
		Client:  server.Client(),
	}

	mark, err := api.GetPolicyMark("noMT")
	if err != nil {
		t.Fatalf("GetPolicyMark returned error: %v", err)
	}
	if mark != 0xffffaa2 {
		t.Fatalf("GetPolicyMark(noMT) = %#x, want 0xffffaa2", mark)
	}

	mark, err = api.GetPolicyMark("Policy0")
	if err != nil {
		t.Fatalf("GetPolicyMark returned error: %v", err)
	}
	if mark != 0xffffaa1 {
		t.Fatalf("GetPolicyMark(Policy0) = %#x, want 0xffffaa1", mark)
	}

	if _, err := api.GetPolicyMark("missing"); err == nil {
		t.Fatalf("GetPolicyMark(missing) should fail for unknown policy")
	}
}
