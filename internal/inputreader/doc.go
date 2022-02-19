// Package inputreader provides an interface to read
// generic data (as in []byte) from various sources such
// as HTTP(S), local file or S3.
//
// These various "backends" are supported by parsing an
// URL and deciding on the method required to read the file
// at hand based on the protocol fration of the URL.
//
// All parameters (such as the URL itself, the body in case of
// HTTP(S) requests etc.) can be templated using standard go
// templates.

package inputreader
