# Playground for joy
Playground for https://github.com/matthewmueller/joy heavily based on https://github.com/golang/playground and https://github.com/gopherjs/gopherjs.github.io/

## Installation
This project is split into 2 parts. The playground and snippet store.

### Playground installation
Make sure to install https://github.com/matthewmueller/joy first, modify the path at https://github.com/tryy3/joyplayground/blob/master/sandbox/playground.go#L168 to where you installed joy.

Once joy is installed, you can build and run the sandbox package. For example navigate to the sandbox folder and run the command 'go build && ./sandbox'

Then you can navigate to http://localhost:5555

The playground will work just fine without the snippet store, but if you want to use the share feature, you will also need to install the snippet store.

### Snippet store installation
You don't need much to get snippet store running, when someone share a playground it will simply create a tmp folder so there is no need for any databases of any sort.

You do need to go get github.com/shurcooL/webdavfs/vfsutil if you haven't already but other than that you can simply build and run the snippet store.

If you want to modify the storage dir or what port snippet store runs on you can use the flags:
```
--storage-dir=""   Storage dir for snippets; if empty, a volatile in-memory store is used.
--http=":8080"     Listen for HTTP connections on this address.
```

If you are running snippet store on a different port or a different host, you will need to modify this line https://github.com/tryy3/joyplayground/blob/master/sandbox/playground.go#L24 on the playground server

## TODO
 * Get docker working
 * Maybe split snippet store and playground into different packages to make installation easier with go get?
 * Decide on v1 or v2 and remove version extension
 * https://github.com/tryy3/joyplayground/blob/master/sandbox/playground.go#L196 maybe edit the default package name?
 * https://github.com/tryy3/joyplayground/blob/master/sandbox/playground.go#L168 detect joy installation?
