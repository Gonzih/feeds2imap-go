package feeds2imap

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"github.com/mmcdole/gofeed"
	"github.com/spf13/viper"
)

type imapClient interface {
	Logout() error
	Create(string) error
	Append(string, []string, time.Time, imap.Literal) error
}

var templateFuncs = template.FuncMap{
	"emptyString": func(s string) bool {
		return len(s) == 0
	},
}

var mailTemplate = template.Must(template.New("mail").
	Funcs(templateFuncs).
	Parse(`<table>
<tbody>
<tr><td>
<a href="{{ .Link }}">{{ .Title }}</a>{{ if .Author | emptyString | not }} | {{ .Author }}{{ end }}
<hr>
</td></tr>
<tr><td>
{{ .Content }}
</td></tr>
</tbody>
</table>`))

type templatePayload struct {
	Link    string
	Title   string
	Author  string
	Content template.HTML
}

func formatLink(rawLink string) string {
	u, err := url.Parse(rawLink)
	if err != nil {
		log.Printf("Error parsing link \"%s\": %s", rawLink, err)
		return rawLink
	}

	if u.Scheme == "" {
		u.Scheme = "http"
	}

	return u.String()
}

func formatAuthor(item *gofeed.Item) string {
	if item.Author != nil {
		return fmt.Sprintf("%s %s", item.Author.Name, item.Author.Email)
	}

	return ""
}

func formatContent(item *gofeed.Item) (string, error) {
	var payload templatePayload

	payload.Author = formatAuthor(item)
	payload.Link = formatLink(item.Link)
	payload.Title = item.Title

	if len(item.Content) > 0 {
		payload.Content = template.HTML(item.Content)
	} else {
		payload.Content = template.HTML(item.Description)
	}

	var buffer bytes.Buffer

	err := mailTemplate.Execute(&buffer, payload)

	if err != nil {
		return "", err
	}

	return buffer.String(), nil
}

func newMessage(item *gofeed.Item, feedTitle string) (bytes.Buffer, error) {
	var b bytes.Buffer

	fromName := feedTitle

	if len(fromName) == 0 {
		fromName = viper.GetString("imap.from.name")
	}

	from := []*mail.Address{
		{
			Name:    fromName,
			Address: viper.GetString("imap.from.email"),
		},
	}
	to := []*mail.Address{
		{
			Name:    viper.GetString("imap.to.name"),
			Address: viper.GetString("imap.to.email"),
		},
	}

	mediaParams := map[string]string{"charset": "utf-8"}

	h := mail.NewHeader()
	h.SetContentType("multipart/alternative", mediaParams)
	h.SetDate(*item.PublishedParsed)
	h.SetAddressList("From", from)
	h.SetAddressList("To", to)
	h.SetSubject(item.Title)

	messageWriter, err := mail.CreateWriter(&b, h)
	defer messageWriter.Close()
	if err != nil {
		return b, err
	}

	htmlHeader := mail.NewTextHeader()
	htmlHeader.SetContentType("text/html", mediaParams)
	htmlWriter, err := messageWriter.CreateSingleText(htmlHeader)
	defer htmlWriter.Close()
	if err != nil {
		return b, err
	}

	content, err := formatContent(item)
	if err != nil {
		return b, err
	}

	io.WriteString(htmlWriter, content)

	return b, nil
}

func newIMAPClient() (*client.Client, error) {
	hostPort := fmt.Sprintf("%s:%d", viper.GetString("imap.host"), viper.GetInt("imap.port"))
	c, err := client.DialTLS(hostPort, nil)
	if err != nil {
		return c, err
	}

	if err := c.Login(viper.GetString("imap.username"), viper.GetString("imap.password")); err != nil {
		return c, err
	}

	if viper.GetBool("debug") {
		log.Println("Logged in to IMAP")
	}

	return c, nil
}

func appendNewItemsVia(items ItemsWithFolders, client imapClient) error {
	for _, entry := range items {
		if entry.Item.PublishedParsed == nil {
			t := time.Now()
			entry.Item.PublishedParsed = &t
		}

		folderName := entry.Folder
		if viper.GetBool("imap.folder.capitalize") {
			folderName = strings.Title(folderName)
		}
		folder := fmt.Sprintf("%s/%s", viper.GetString("imap.folder.prefix"), folderName)

		_ = client.Create(folder)

		msg, err := newMessage(entry.Item, entry.FeedTitle)
		if err != nil {
			return err
		}

		if viper.GetBool("debug") {
			log.Printf("Appending item to %s", folder)
		}

		literal := bytes.NewReader(msg.Bytes())
		err = client.Append(folder, []string{}, *entry.Item.PublishedParsed, literal)
		if err != nil {
			return err
		}
	}

	return nil
}

// AppendNewItemsViaIMAP puts items in to corresponding imap folders
func AppendNewItemsViaIMAP(items ItemsWithFolders) error {
	client, err := newIMAPClient()
	if err != nil {
		return err
	}
	defer client.Logout()

	return appendNewItemsVia(items, client)
}
