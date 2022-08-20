package webhook

import (
	"fmt"
	"strings"

	"github.com/ecnepsnai/discord"
)

var (
	ServerName  string
	ServerColor uint32 = 0x05b2f7
)

func PostModcall(character string, area string, reason string) error {
	e := discord.Embed{
		Title:       fmt.Sprintf("%v sent a modcall in %v.", character, area),
		Description: reason,
		Color:       ServerColor,
	}
	p := discord.PostOptions{
		Username: ServerName,
		Embeds:   []discord.Embed{e},
	}
	err := discord.Post(p)
	return err
}

func PostReport(name string, contents string) error {
	c := strings.NewReader(contents)
	f := discord.FileOptions{
		FileName: name,
		Reader:   c,
	}
	p := discord.PostOptions{
		Username: ServerName,
	}
	err := discord.UploadFile(p, f)
	return err
}
