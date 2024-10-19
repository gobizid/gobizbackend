package watoken

import (
	"fmt"
	"testing"

	"github.com/gocroot/config"
)

func TestEncode(t *testing.T) {
	privkey := config.PRIVATEKEY
	str, _ := EncodeforHours("6282184952582", "Ahmad Rifki Ayala", privkey, 43830)
	println(str)
	//atr, _ := DecodeGetId("", str)
	//println(atr)
	payload, _ := Decode(config.PUBLICKEY, "v4.public.eyJhbGlhcyI6IkFobWFkIFJpZmtpIEF5YWxhIiwiZXhwIjoiMjAyOS0xMC0xOVQxNzoyMTo0OCswNzowMCIsImlhdCI6IjIwMjQtMTAtMTlUMTE6MjE6NDgrMDc6MDAiLCJpZCI6IjYyODIxODQ5NTI1ODIiLCJuYmYiOiIyMDI0LTEwLTE5VDExOjIxOjQ4KzA3OjAwIn0SkHj-0wE0bb3V5BAiiIpqnvZ3Byb1ZHlG8GNVLvRufCEAxDoz75_pwCC2OrwjFcqptw3I57qo5aWpSINBKVgD")
	println(fmt.Sprintf("%v", payload))
}
