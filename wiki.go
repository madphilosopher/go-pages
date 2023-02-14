package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var md = goldmark.New(goldmark.WithExtensions(extension.Linkify, extension.GFM))

var baseTemplate = template.New("wiki")

func loadTemplates(templateDirectory string) error {
	// Load base templates for reusing
	foundTemplates := make([]string, 0)
	contents, err := ioutil.ReadDir(templateDirectory)
	if err != nil {
		return err
	}
	for _, fileinfo := range contents {
		if path.Ext(fileinfo.Name()) == ".tpl" {
			foundTemplates = append(foundTemplates, path.Join(templates, fileinfo.Name()))
		}
	}

	_, err = baseTemplate.ParseFiles(foundTemplates...)
	if err != nil {
		return err
	}

	return nil
}

// Node holds a Wiki node.
type Node struct {
	Title    string
	Basepath string
	Path     string
	File     string
	Content  string
	Template string
	Revision string
	Bytes    []byte
	Dirs     []*Directory
	Log      []*Log
	Markdown template.HTML

	Edit      bool // Edit mode
	Revisions bool // Show revisions
	AskDelete bool // Delete mode
	Author    string
	Changelog string
}

// Directory lists nodes.
type Directory struct {
	Path   string
	Name   string
	Active bool
}

// Log is an event in the past.
type Log struct {
	Hash    string
	Message string
	Time    string
	Link    bool
}

func (node *Node) isHead() bool {
	return len(node.Log) > 0 && node.Revision == node.Log[0].Hash
}

// ToMarkdown processes the node contents.
func (node *Node) ToMarkdown() {
	var source = node.Bytes
	var buf bytes.Buffer
	if err := md.Convert(source, &buf); err != nil {
		panic(err)
	}

	node.Markdown = template.HTML(buf.String())
}

func wikiHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/favicon.ico" {
		return
	}
	// Params
	content := r.FormValue("content")
	changelog := r.FormValue("msg")
	author := r.FormValue("author")
	reset := r.FormValue("revert")
	revision := r.FormValue("revision")

	// Default to index page on trailing slash
	if r.URL.Path[len(r.URL.Path)-1] == '/' {
		r.URL.Path += "index"
	}
	filePath := fmt.Sprintf("%s%s.md", directory, r.URL.Path)
	node := &Node{
		File:     r.URL.Path[1:] + ".md",
		Path:     r.URL.Path,
		Title:    r.URL.Path[1:] + " – " + title,
		Basepath: strings.TrimSuffix(basepath, "/"), // we do not want basepath to end with a /
	}
	node.Revisions = parseBool(r.FormValue("revisions"))
	node.Edit = parseBool(r.FormValue("edit"))
	node.AskDelete = parseBool(r.FormValue("askdelete"))

	if cookie, err := r.Cookie("author"); err == nil {
		node.Author = cookie.Value
	}
	if node.Author == "" {
		node.Author = "Unknown"
	}

	// Delete if needed
	deleteNow := parseBool(r.FormValue("delete"))
	if deleteNow {
		// Delete file
		file := r.URL.Path
		changelog := fmt.Sprintf("Delete %s", node.File)
		node.GitRemove().GitCommit(changelog, author)
		// Move node path one level up and redirect
		var location string
		if file[len(file)-1] == '/' {
			location = strings.TrimRight(file, "/")
		} else {
			file = file[:strings.LastIndexAny(file, "/")+1]
		}
		http.Redirect(w, r, location, http.StatusTemporaryRedirect)
		return
	}

	node.Dirs = listDirectories(r.URL.Path)

	// We have content, update
	if content != "" && changelog != "" && author != "" {
		node.Author = author
		bytes := []byte(content)
		err := writeFile(bytes, filePath)
		if err != nil {
			log.Printf("Could not write to file %q, error: %v", filePath, err)
		} else {
			// Wrote file, commit
			node.Bytes = bytes
			node.GitAdd().GitCommit(changelog, author).GitLog()
			node.ToMarkdown()
		}
	} else if reset != "" {
		// Reset to revision
		node.Revision = reset
		node.GitRevert().GitCommit("Reverted to: "+node.Revision, author)
		node.Revision = ""
		node.GitShow().GitLog()
		node.ToMarkdown()
	} else {
		// Show specific revision
		node.Revision = revision
		node.GitShow().GitLog()

		createNew := len(node.Bytes) == 0
		node.Edit = node.Edit || createNew

		changelogPageName := strings.TrimLeft(node.Path, "/")
		if changelogPageName == "" {
			changelogPageName = "index page"
		}
		node.Changelog = fmt.Sprintf("Edit %s", changelogPageName)
		if createNew {
			node.Changelog = fmt.Sprintf("Create %s", changelogPageName)
		}

		if node.Edit {
			node.Content = string(node.Bytes)
			node.Template = "edit.tpl"
		} else {
			node.ToMarkdown()
		}
	}
	renderTemplate(w, node)
}

func writeFile(bytes []byte, entry string) error {
	err := os.MkdirAll(path.Dir(entry), 0777)
	if err == nil {
		return ioutil.WriteFile(entry, bytes, 0644)
	}
	return err
}

func setCookie(w http.ResponseWriter, name, value string) {
	expiration := time.Now().AddDate(1, 0, 0)
	cookie := http.Cookie{Name: name, Value: value, Expires: expiration}
	http.SetCookie(w, &cookie)
}

func renderTemplate(w http.ResponseWriter, node *Node) {
	// Set cookies
	setCookie(w, "author", node.Author)

	// Clone base template
	t, err := baseTemplate.Clone()
	if err != nil {
		log.Fatalln("Could not clone baseTemplate:", err)
	}

	// Build content template
	if node.Markdown != "" {
		tpl := "{{ template \"header\" . }}"

		// Show revisions
		if node.Revisions {
			tpl += "{{ template \"revisions\" . }}"
		}

		if !node.isHead() && node.Revision != "" {
			tpl += "{{ template \"revision\" . }}"
		}
		// Add node
		tpl += "{{ template \"node\" . }}"

		// Footer
		tpl += "{{ template \"footer\" . }}"
		if t, err = t.Parse(tpl); err != nil {
			log.Fatalf("Could not parse template %q: %v", tpl, err)
		}
		// Execute
		err = t.Execute(w, node)
	} else if node.Template != "" {
		err = t.ExecuteTemplate(w, node.Template, node)
	}
	if err != nil {
		log.Fatal("Could not execute template:", err)
	}

}
