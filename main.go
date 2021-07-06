package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"git.sr.ht/~adnano/go-gemini"
)

type TinyLog struct {
	Author  string
	Avatar  string
	Link    string
	Entries []*TlEntry
}

type TlEntry struct {
	Published time.Time
	Content   string
}

var Tl TinyLog

func main() {
	if os.Getenv("QUERY_STRING") == "" {
		fmt.Print("10\ttext/gemtext\r\n")
		fmt.Print("Enter the URL to test: ")
		os.Exit(0)
	}

	re := regexp.MustCompile(`(?im)^\w+$`)
	if re.MatchString(os.Getenv("QUERY_STRING")) == false {
		fmt.Print("20 text/gemini\r\n")
		fmt.Println("Wrong user name", os.Getenv("QUERY_STRING"))
		fmt.Print("\r\n")
		os.Exit(0)
	}
	link := "gemini://station.martinrue.com/" + os.Getenv("QUERY_STRING")
	e := Tl.generateFromStationPage(link)
	if e != nil {
		fmt.Print("20 text/gemini\r\n")
		fmt.Println("Error retrieving from station:", e)
		fmt.Print("\r\n")
		os.Exit(0)
	}

	tl := Tl.generateTinyLog()
	fmt.Print("20 text/gemini\r\n")
	fmt.Print(tl)
	fmt.Print("\n")
	os.Exit(0)
}

func (Tl *TinyLog) generateFromStationPage(link string) error {
	Tl.Link = link
	content, err := getStationPage(Tl.Link)
	if err != nil {
		return err
	}
	return Tl.parseStationPage(content)
}

func (Tl *TinyLog) generateTinyLog() string {
	tinylog := "# " + Tl.Author + "'s TinyLog - Generated from station\n\n"
	tinylog = tinylog + "Author: @" + Tl.Author + "\n"
	if Tl.Avatar != "" {
		tinylog = tinylog + "Avatar: " + Tl.Avatar + "\n"
	}

	tinylog = tinylog + "\n"

	for i := 0; i < len(Tl.Entries); i++ {
		tinylog = tinylog + "## " + Tl.Entries[i].Published.Format("Mon 02 Jan 2006 15:04 -0700") + "\n" + Tl.Entries[i].Content + "\n\n"
	}

	return tinylog
}

func (Tl *TinyLog) parseStationPage(content string) error {
	lines := strings.Split(content, "\n")

	foundHeader := false
	startLogs := 0
	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "### Logs") {
			startLogs = i + 1
		} else if foundHeader == false && strings.HasPrefix(lines[i], "###") {
			h := strings.Split(lines[i], " ")
			Tl.Author = h[2]
			Tl.Avatar = h[1]
			foundHeader = true
		}
	}

	// Create slice:
	entries := strings.Split(strings.Join(lines[startLogs:], "\n"), "\n\n")

	var TlEntries []*TlEntry
	for i := 0; i < len(entries); i++ {
		//fmt.Println(entries[i])
		e := strings.Split(strings.TrimSpace(entries[i]), "\n")
		if len(e) >= 3 {
			var entry TlEntry
			entry.Published = parseEntryDate(e[2])
			entry.Content = strings.TrimSpace(e[1])
			TlEntries = append(TlEntries, &entry)
		}
	}

	Tl.Entries = TlEntries
	return nil
}

func parseEntryDate(footer string) time.Time {
	f := strings.Split(footer, "·")
	relatedDate := strings.Split(strings.TrimSpace(f[len(f)-1]), " ")

	num, _ := strconv.Atoi(strings.TrimSpace(relatedDate[0]))
	d := time.Duration(-num)

	tn := time.Now()
	if strings.Contains(strings.TrimSpace(relatedDate[1]), "second") {
		tn = tn.Add(d * time.Second)
	} else if strings.Contains(strings.TrimSpace(relatedDate[1]), "minute") {
		tn = tn.Add(d * time.Minute)
	} else if strings.Contains(strings.TrimSpace(relatedDate[1]), "hour") {
		tn = tn.Add(d * time.Hour)
	} else if strings.Contains(strings.TrimSpace(relatedDate[1]), "day") {
		tn = tn.Add(d * time.Hour * 24)
	} else if strings.Contains(strings.TrimSpace(relatedDate[1]), "week") {
		tn = tn.Add(d * time.Hour * 24 * 7)
	} else if strings.Contains(strings.TrimSpace(relatedDate[1]), "month") {
		tn = tn.Add(d * time.Hour * 24 * 30)
	} else if strings.Contains(strings.TrimSpace(relatedDate[1]), "year") {
		tn = tn.Add(d * time.Hour * 24 * 365)
	}

	return tn
}

func getStationPage(link string) (string, error) {
	Tl.Link = link
	gemclient := &gemini.Client{}
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(2)*time.Second)

	response, err := gemclient.Get(ctx, link)

	if err != nil {
		return "", fmt.Errorf("Error retrieving content from %v", link)
	}
	defer response.Body.Close()

	// TODO: Add an option to accept gemini feeds with expired certificate.
	// TODO: Add possibility to validate certs?
	if respCert := response.TLS().PeerCertificates; len(respCert) > 0 && time.Now().After(respCert[0].NotAfter) {
		return "", fmt.Errorf("Invalid certificate for capsule:", link)
	}

	c, err := io.ReadAll(response.Body)
	if err != nil || len(c) == 0 {
		return "", fmt.Errorf("Couldn't read response from tinylogs", link)
	}

	return string(c), nil
}
