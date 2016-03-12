// Package twilio implements the github.com/itsabot/abot/interface/sms/driver
// interface.
package twilio

import (
	"encoding/xml"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/itsabot/abot/core"
	"github.com/itsabot/abot/shared/datatypes"
	"github.com/itsabot/abot/shared/interface/sms"
	"github.com/itsabot/abot/shared/interface/sms/driver"
	"github.com/itsabot/abot/shared/log"
	"github.com/labstack/echo"
	"github.com/subosito/twilio"
)

type drv struct{}

func (d *drv) Open(name string, e *echo.Echo) (driver.Conn, error) {
	auth := strings.Split(name, ":")
	c := conn(*twilio.NewClient(auth[0], auth[1], nil))
	hm := dt.NewHandlerMap([]dt.RouteHandler{
		{
			// Path is prefixed by "twilio" automatically. Thus the
			// path below becomes "/twilio"
			Path:    "/",
			Method:  echo.POST,
			Handler: handlerTwilio,
		},
	})
	hm.AddRoutes("twilio", e)
	return &c, nil
}

func init() {
	sms.Register("twilio", &drv{})
}

type conn twilio.Client

// Send an SMS using a Twilio client to a specific phone number in the following
// valid international format ("+13105555555").
func (c *conn) Send(from, to, msg string) error {
	params := twilio.MessageParams{Body: msg}
	_, _, err := c.Messages.Send(from, to, params)
	return err
}

// Close the connection, but since Twilio connections are open as needed, there
// is nothing for us to close here. Return nil.
func (c *conn) Close() error {
	return nil
}

// phoneRegex determines whether a string is a phone number
var phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)

type Phone string

// Valid determines the validity of a phone number according to Twilio's
// expectations for the number's formatting. If valid, error will be nil. If
// invalid, the returned error will contain the reason.
func (p Phone) Valid() (valid bool, err error) {
	if len(p) < 10 || len(p) > 20 || !phoneRegex.MatchString(string(p)) {
		return false, errors.New("invalid phone number format: must have E.164 formatting")
	}
	if len(p) == 11 && p[0] != '1' {
		return false, errors.New("unsupported international number")
	}
	if p[0] != '+' {
		return false, errors.New("first character in phone number must be +")
	}
	if len(p) == 12 && p[1] != '1' {
		return false, errors.New("unsupported international number")
	}
	return true, nil
}

type TwilioResp struct {
	XMLName xml.Name `xml:"Response"`
	Message string
}

// handlerTwilio responds to SMS messages sent through Twilio. Unlike other
// handlers, we process internal errors without returning here, since any errors
// should not be presented directly to the user -- they should be "humanized"
func handlerTwilio(c *echo.Context) error {
	c.Set("cmd", c.Form("Body"))
	c.Set("flexid", c.Form("From"))
	c.Set("flexidtype", 2)
	ret, _, err := core.ProcessText(c)
	if err != nil {
		log.Debug("couldn't process text", err)
		ret = "Something went wrong with my wiring... I'll get that fixed up soon."
	}
	/*
		// TODO
		if err = ws.NotifySockets(c, uid, c.Form("Body"), ret); err != nil {
			return core.JSONError(err)
		}
	*/
	var resp TwilioResp
	if len(ret) == 0 {
		resp = TwilioResp{}
	} else {
		resp = TwilioResp{Message: ret}
	}
	if err = c.XML(http.StatusOK, resp); err != nil {
		return core.JSONError(err)
	}
	return nil
}
