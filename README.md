tilesetter
==========

Do you deal with Pivotal Tiles (i.e. for a Pivotal CloudFoundry
deployment)?  Ever wonder what's inside of them?  Ever wondered
what BOSH templates a given release property feeds?

No?  Then this software is most definitely not for you.

Otherwise, you just might find _tilesetter_ interesting.

What is it?
-----------

Tilesetter is a web-based application that lets you analyze
Pivotal tiles, by breaking them down into their component BOSH
releases, jobs, and templates.  You can then browse these bits and
pieces, to see what properties are defined, and how they are used
in the job templates.

The web UI also allows you to search for manifest properties.
Along with properties whose names match your search criteria, you
will also get to see all of the related templates that appear to
use those properties.  This can be quite useful when debugging
failing deployments.

How does it work?
-----------------

Ultimately, Pivotal tiles are just ZIP archives of some metadata,
and some release tarballs.  These release tarballs, in turn, are
just some release-specific metadata and some job tarballs.  These
job tarballs are just some metadata (the job _spec_ file) and the
template sources.

It's basically archives all the way down.

`tilesetter` unravels this structure and extracts the relevant
bits (tile &rarr; release &rarr; job) into a format that it can
then search and render in a browser.

How do I run it?
----------------

First, you need to install it.  Assuming your `$GOPATH` is set up:

```
go install github.com/jhunt/go-tilesetter
```

To run it, assuming that `$GOPATH/bin` is in your `$PATH`:

```
go-tilesetter api $GOPATH/src/github/jhunt/go-tilesetter/web/ui
```

That will start up a daemon listening on port 5001.  You can
change this by passing a third argument to the `go-tilesetter`
command, containing the IP:port to bind.

From there, you'll need to upload one or more tiles.  Right now,
there isn't a way to do this from within the web UI, so we get to
use curl!

```
curl -XPOST http://localhost:5001/upload -Ftile=@/path/to/tile
```

Note that it may take a while to upload, especially if its a large
tile like the PCF Elastic Runtime (~4-5G).  When it's done, curl
should print out something like `{"ok":"uploaded"}`

Now, you can visit http://127.0.0.1:5001 in your browser and
explore to your heart's content!

Any advice?
-----------

1. Keep in mind that this is **BETA** software; it's bound to have
   bugs, leak memory like a sieve, chew your CPU, etc.
2. For your own sanity, don't upload too many tiles to a single
   instance of tilesetter.  For starters, it slows the app down.
   For finishers, it slows you down when you search.  Do you have
   any idea how many releases define a `cf.admin_username`
   property?
3. The template-search algorithm is heuristic.  It may not catch
   all uses of a set of matching properties.  When in doubt,
   browse the job and read all the templates.
