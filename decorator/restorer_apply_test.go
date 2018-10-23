package decorator

import (
	"bytes"
	"go/format"
	"testing"

	"github.com/dave/dst"
	"github.com/dave/dst/dstutil"
)

func TestRestorerApply(t *testing.T) {
	tests := []struct {
		skip, solo bool
		name       string
		code       string
		f          func(*dst.File)
		pre, post  func(*dstutil.Cursor) bool
		expect     string
	}{
		{
			name: "node-reuse",
			code: `package a

var i /*a*/ int`,
			f: func(f *dst.File) {
				gd := dst.Clone(f.Decls[0]).(*dst.GenDecl)
				gd.Decs.Space = dst.NewLine
				gd.Specs[0].(*dst.ValueSpec).Names[0].Name = "j"
				gd.Specs[0].(*dst.ValueSpec).Decs.Names.Replace("/*b*/")
				f.Decls = append(f.Decls, gd)
			},
			expect: `package a

var i /*a*/ int
var j /*b*/ int`,
		},
		{
			name: "simple",
			code: `package a

				var i int`,
			pre: func(c *dstutil.Cursor) bool {
				switch n := c.Node().(type) {
				case *dst.Ident:
					if n.Name == "i" {
						n.Name = "j"
					}
				}
				return true
			},
			expect: `package a

				var j int`,
		},
	}
	var solo bool
	for _, test := range tests {
		if test.solo {
			solo = true
			break
		}
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if solo && !test.solo {
				t.Skip()
			}
			if test.skip {
				t.Skip()
			}

			// format code and check it hasn't changed
			bCode, err := format.Source([]byte(test.code))
			if err != nil {
				t.Fatal(err)
			}
			if normalize(string(bCode)) != normalize(test.code) {
				t.Fatalf("code changed after gofmt. before: \n%s\nafter:\n%s", test.code, string(bCode))
			}
			// use the formatted version (correct indents etc.)
			test.code = string(bCode)

			// format expect and check it hasn't changed
			bExpect, err := format.Source([]byte(test.expect))
			if err != nil {
				t.Fatal(err)
			}
			if normalize(string(bExpect)) != normalize(test.expect) {
				t.Fatalf("expect changed after gofmt. before: \n%s\nafter:\n%s", test.expect, string(bExpect))
			}
			// use the formatted version (correct indents etc.)
			test.expect = string(bExpect)

			file, err := Parse(test.code)
			if err != nil {
				t.Fatal(err)
			}

			if test.f != nil {
				test.f(file)
			}

			if test.pre != nil || test.post != nil {
				file = dstutil.Apply(file, test.pre, test.post).(*dst.File)
			}

			restoredFset, restoredFile := Restore(file)

			buf := &bytes.Buffer{}
			if err := format.Node(buf, restoredFset, restoredFile); err != nil {
				t.Fatal(err)
			}

			if buf.String() != test.expect {
				t.Errorf("expected: %s\ngot: %s", test.expect, buf.String())
				//t.Errorf("diff: %s", diff.LineDiff(test.expect, buf.String()))
			}
		})
	}
}
