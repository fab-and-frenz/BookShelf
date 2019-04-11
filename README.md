# BookShelf

A place to put your books.

BookShelf allows you to serve up an entire library of ebooks through http, so they can be read through the web browser of any device.

## File Support

Below is a table showing the current support for different ebook filetypes.
Everything listed in the table is planned to be added in the future.
You may request other formats by opening an issue in Github!

Format | Viewing | Annotations
-------|---------|------------
Cbz    | no      | no
Cbt    | no      | no
Cbr    | no      | no
Epub   | yes     | no
Mobi   | no      | no
Pdf    | yes     | no

## Build & Run

You must have Golang, Sass, and MongoDB installed.
Once these prerequisites have been met, build and run BookShelf with the following commands:

```sh
# get the source code
git clone github.com/fab-and-frenz/bookshelf

# go to the root directory
cd bookshelf

# compile the code
go build

# compile the scss
sass html/scss:html/css

# run bookshelf with the certificate `cert.pem` and private key `privkey.pem`
./bookshelf -cert cert.pem -privkey privkey.pem
```
