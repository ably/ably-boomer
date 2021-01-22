package ablyboomer

import (
	"bytes"
	"strings"
	"text/template"
)

// ChannelTemplateData is used to render templated lists of channels with the
// number of the current user.
//
// Typical usage is to either generate "personal" channels that are unique to
// each user like:
//
//     personal-{{ printf "%04d" .UserNumber }}
//
// or "sharded" channels so that users are spread amongst a fixed number of
// channels like:
//
//     sharded-{{ mod .UserNumber 5 }}
//
type ChannelTemplateData struct {
	UserNumber int64
}

// channelFuncs are the functions available to channel list templates.
var channelFuncs = template.FuncMap{
	"mod": channelFuncMod,
}

// channelFuncMod is a channel template function that returns the given number
// modulo the given modulus, useful when generating sharded channel names.
func channelFuncMod(num, modulus int64) int64 {
	return num % modulus
}

// renderChannels renders the given templated list of channels using the given
// user number.
func renderChannels(tmpl *template.Template, userNum int64) []string {
	var buf bytes.Buffer
	tmpl.Execute(&buf, &ChannelTemplateData{UserNumber: userNum})
	return strings.Split(buf.String(), ",")
}
