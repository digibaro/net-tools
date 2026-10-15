package tools

import (
	"testing"

	"github.com/digibaro/net-tools-docker/apps/api/internal/config"
)

func TestRequireHelpers(t *testing.T) {
	d := Deps{Config: config.Config{}}
	if ok, r := RequireLAN(d); ok || r != ReasonRequiresLANMode {
		t.Errorf("lan off: %v %s", ok, r)
	}
	d.Config.Features.LAN = true
	if ok, _ := RequireLAN(d); !ok {
		t.Error("lan on")
	}
	if ok, r := RequireFlag(false); ok || r != ReasonDisabledByAdmin {
		t.Errorf("flag off: %v %s", ok, r)
	}
}
