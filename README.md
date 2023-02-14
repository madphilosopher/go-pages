# go-pages

A wiki tool built on golang with git as the storage back-end. Content is formatted in [commonmark syntax](https://spec.commonmark.org/0.30/). The wiki is rendered with go templates, [bootstrap](http://getbootstrap.com) css and [highlightjs](https://highlightjs.org) for code highlighting but doesn't depend on any CDN. This project was forked from [aspic/g-wiki](https://github.com/aspic/g-wiki).

The [madphilosopher fork](https://github.com/madphilosopher/go-pages) adds the following features/changes:

* Individual wiki pages get their HTML `<title>` from the node's filename

## Using

Available command line flags are:

* `--address=:8080` *(in the format ip:port, empty ip binds to all IP addresses)*
* `--dir=files` *(data directory has to be an initialized git repository!)*
* `--title=CoolWiki` *(global wiki title)*
* `--basepath=/wiki/` *(base path for reverse proxy web applications)*

## Extensions

The goldmark rendering engine supports extensions which can be found here:

* [https://github.com/yuin/goldmark/#built-in-extensions](https://github.com/yuin/goldmark/#built-in-extensions)
* [https://github.com/yuin/goldmark/#extensions](https://github.com/yuin/goldmark/#extensions)

## Example screenshot

![Screenshot](static/screenshots/screenshot1.jpg)
