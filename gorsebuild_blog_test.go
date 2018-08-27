package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePost(t *testing.T) {
	tests := []struct {
		Name    string
		Content string
		Output  Post
	}{
		{
			"perl-and-character-encoding.md",
			`# Perl and character encoding

META description Perl's approach to character encoding presents challenges. I
describe it and offer tips for working with it.
META pubdate 2017-09-23 12:40:29

Try this program:

<pre><code>
use strict;
use warnings;

use Data::Dumper qw( Dumper );

# Use double quotes for strings. This enables showing octal and the \x{} output
# we rely on below.
`,
			Post{
				Title:       "Perl and character encoding",
				Description: "Perl's approach to character encoding presents challenges. I describe it and offer tips for working with it.",
				PubDate: func() time.Time {
					d, err := time.ParseInLocation(
						"2006-01-02 15:04:05",
						"2017-09-23 12:40:29",
						time.Local,
					)
					require.NoError(t, err)
					return d
				}(),
				URI: "https://blog.summercat.com/perl-and-character-encoding.html",
			},
		},
	}

	for _, test := range tests {
		buf := bytes.NewBufferString(test.Content)
		post, err := parsePost(test.Name, buf)
		require.NoError(t, err)
		assert.Equal(t, test.Output, post)
	}
}
