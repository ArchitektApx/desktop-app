package commands

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/ivpn/desktop-app-cli/flags"
	"github.com/ivpn/desktop-app-daemon/service"
)

type CmdWireGuard struct {
	flags.CmdInfo
	state            bool
	regenerate       bool
	rotationInterval int
}

func (c *CmdWireGuard) Init() {
	c.Initialize("wgkeys", "WireGuard keys management")
	c.BoolVar(&c.state, "status", false, "(default) Show WireGuard configuration")
	c.IntVar(&c.rotationInterval, "rotation_interval", 0, "DAYS", "Set WireGuard keys rotation interval. [1-30] days (default = 7 days)")
	c.BoolVar(&c.regenerate, "regenerate", false, "Regenerate WireGuard keys")
}
func (c *CmdWireGuard) Run() error {
	if c.rotationInterval < 0 || c.rotationInterval > 30 {
		fmt.Println("Error: keys rotation interval should be in diapasone [1-30] days")
		return flags.BadParameter{}
	}

	defer func() {
		helloResp := _proto.GetHelloResponse()
		if len(helloResp.Session.Session) == 0 {
			fmt.Println(service.ErrorNotLoggedIn{})

			PrintTips([]TipType{TipLogin})
		}
	}()

	resp, err := _proto.SendHello()
	if err != nil {
		return err
	}
	if len(resp.DisabledFunctions.WireGuardError) > 0 {
		return fmt.Errorf("WireGuard functionality disabled:\n\t" + resp.DisabledFunctions.WireGuardError)
	}

	if c.regenerate {
		fmt.Println("Regenerating WG keys...")
		if err := c.generate(); err != nil {
			return err
		}
	}

	if c.rotationInterval > 0 {
		interval := time.Duration(time.Hour * 24 * time.Duration(c.rotationInterval))
		fmt.Printf("Changing WG keys rotation interval to %v ...\n", interval)
		if err := c.setRotateInterval(int64(interval / time.Second)); err != nil {
			return err
		}
	}

	if err := c.getState(); err != nil {
		return err
	}

	return nil
}

func (c *CmdWireGuard) generate() error {
	return _proto.WGKeysGenerate()
}

func (c *CmdWireGuard) setRotateInterval(interval int64) error {
	return _proto.WGKeysRotationInterval(interval)
}

func (c *CmdWireGuard) getState() error {
	resp, err := _proto.SendHello()
	if err != nil {
		return err
	}

	if len(resp.Session.Session) == 0 {
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintln(w, fmt.Sprintf("Local IP:\t%v", resp.Session.WgLocalIP))
	fmt.Fprintln(w, fmt.Sprintf("Public KEY:\t%v", resp.Session.WgPublicKey))
	fmt.Fprintln(w, fmt.Sprintf("Generated:\t%v", time.Unix(resp.Session.WgKeyGenerated, 0)))
	fmt.Fprintln(w, fmt.Sprintf("Rotation interval:\t%v", time.Duration(time.Second*time.Duration(resp.Session.WgKeysRegenInerval))))
	w.Flush()

	return nil
}
