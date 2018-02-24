// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
In the absence of any formal way to specify interfaces in JavaScript,
here's a skeleton implementation of a playground transport.

        function Transport() {
                // Set up any transport state (eg, make a websocket connection).
                return {
                        Run: function(body, output, options) {
                                // Compile and run the program 'body' with 'options'.
                // Call the 'output' callback to display program output.
                                return {
                                        Kill: function() {
                                                // Kill the running program.
                                        }
                                };
                        }
                };
        }

    // The output callback is called multiple times, and each time it is
    // passed an object of this form.
        var write = {
                Kind: 'string', // 'start', 'stdout', 'stderr', 'end'
                Body: 'string'  // content of write or end status message
        }

    // The first call must be of Kind 'start' with no body.
    // Subsequent calls may be of Kind 'stdout' or 'stderr'
    // and must have a non-null Body string.
    // The final call should be of Kind 'end' with an optional
    // Body string, signifying a failure ("killed", for example).

    // The output callback must be of this form.
    // See PlaygroundOutput (below) for an implementation.
        function outputCallback(write) {
        }
*/

var consoleLogs = []

function HTTPTransport() {
    'use strict';

    // TODO(adg): support stderr

    function playback(output, events) {
        var timeout;
        output({Kind: 'start'});
        function next() {
            if (!events || events.length === 0) {
                output({Kind: 'end'});
                return;
            }
            var e = events.shift();
            if (e.Delay === 0) {
                output({Kind: e.Kind, Body: e.Message});
                next();
                return;
            }
            timeout = setTimeout(function() {
                output({Kind: e.Kind, Body: e.Message});
                next();
            }, e.Delay / 1000000);
        }
        next();
        return {
            Stop: function() {
                clearTimeout(timeout);
            }
        }
    }

    function error(output, msg) {
        output({Kind: 'start'});
        output({Kind: 'stderr', Body: msg});
        output({Kind: 'end'});
    }

    var seq = 0;
    return {
        Run: function(body, output, options) {
            consoleLogs = []
            seq++;
            var cur = seq;
            var playing;
            $.ajax(options.API + '/compile', {
                type: 'POST',
                data: JSON.stringify({'version': 2, 'body': body}),
                dataType: 'json',
                contentType: 'application/json',
                success: function(data) {
                    if (seq != cur) return;
                    if (!data) return;
                    if (playing != null) playing.Stop();
                    if (data.Errors) {
                        error(output, data.Errors);
                        return;
                    }
                    playing = playback(output, data.Events);
                },
                error: function(e) {
                    error(output, 'Error communicating with remote server.');
                }
            });
            return {
                Kill: function() {
                    if (playing != null) playing.Stop();
                    output({Kind: 'end', Body: 'killed'});
                }
            };
        }
    };
}

function SocketTransport() {
    'use strict';

    var id = 0;
    var outputs = {};
    var started = {};
    var websocket = new WebSocket('ws://' + window.location.host + '/socket');

    websocket.onclose = function() {
        console.log('websocket connection closed');
    }

    websocket.onmessage = function(e) {
        var m = JSON.parse(e.data);
        var output = outputs[m.Id];
        if (output === null)
            return;
        if (!started[m.Id]) {
            output({Kind: 'start'});
            started[m.Id] = true;
        }
        output({Kind: m.Kind, Body: m.Body});
    }

    function send(m) {
        websocket.send(JSON.stringify(m));
    }

    return {
        Run: function(body, output, options) {
            var thisID = id+'';
            id++;
            outputs[thisID] = output;
            send({Id: thisID, Kind: 'run', Body: body, Options: options});
            return {
                Kill: function() {
                    send({Id: thisID, Kind: 'kill'});
                }
            };
        }
    };
}

function PlaygroundOutput(el, fileEl) {
    'use strict';

    function output(write) {
        if (write.Kind == 'file') {
            if (typeof fileEl !== 'undefined') {
                var m = write.Body
                m = m.replace(/&/g, '&amp;');
                m = m.replace(/</g, '&lt;');
                m = m.replace(/>/g, '&gt;');
                fileEl.html(m)
            }
            
            var e = write.Body
            e = e.replace("console.log", "window.joyOutput.log")
            eval(e)
            return
        }

        if (write.Kind == 'start') {
            el.innerHTML = '';
            return;
        }

        var cl = 'system';
        if (write.Kind == 'stdout' || write.Kind == 'stderr')
            cl = write.Kind;

        var m = write.Body;
        if (write.Kind == 'end') {
            if ($(el).find(".system").length >= 1) return
            m = '\nProgram exited' + (m?(': '+m):'.');
        }

        if (m.indexOf('IMAGE:') === 0) {
            // TODO(adg): buffer all writes before creating image
            var url = 'data:image/png;base64,' + m.substr(6);
            var img = document.createElement('img');
            img.src = url;
            el.appendChild(img);
            return;
        }

        // ^L clears the screen.
        var s = m.split('\x0c');
        if (s.length > 1) {
            el.innerHTML = '';
            m = s.pop();
        }

        m = m.replace(/&/g, '&amp;');
        m = m.replace(/</g, '&lt;');
        m = m.replace(/>/g, '&gt;');

        var needScroll = (el.scrollTop + el.offsetHeight) == el.scrollHeight;

        var span = document.createElement('span');
        span.className = cl;
        span.innerHTML = m;
        el.appendChild(span);

        if (needScroll)
            el.scrollTop = el.scrollHeight - el.offsetHeight;
    }

    if (typeof window.joyOutput === 'undefined') window.joyOutput = {}
    window.joyOutput.log = function() {
        console.log.apply(console.log, arguments)
        var m = ""
        for (var o of arguments) {
            m += o + " "
        }
        consoleLogs.push(m)

        output({Kind: 'start'});
        for (var msg of consoleLogs) {
            output({Kind: 'stdout', Body: msg});
        }
        output({Kind: 'end'});
    }

    return output
}

(function() {
    function lineHighlight(error) {
        var regex = /prog.go:([0-9]+)/g;
        var r = regex.exec(error);
        while (r) {
            $(".lines div").eq(r[1]-1).addClass("lineerror");
            r = regex.exec(error);
        }
    }
    function highlightOutput(wrappedOutput) {
        return function(write) {
            if (write.Body) lineHighlight(write.Body);
            wrappedOutput(write);
        }
    }
    function lineClear() {
        $(".lineerror").removeClass("lineerror");
    }

    // opts is an object with these keys
    //    codeEl - code editor element
    //    outputEl - program output element
    //    runEl - run button element
    //    fmtEl - fmt button element (optional)
    //    fmtImportEl - fmt "imports" checkbox element (optional)
    //    shareEl - share button element (optional)
    //    shareURLEl - share URL text input element (optional)
    //    shareRedirect - base URL to redirect to on share (optional)
    //    toysEl - toys select element (optional)
    //    enableHistory - enable using HTML5 history API (optional)
    //    transport - playground transport to use (default is HTTPTransport)
    //    enableShortcuts - whether to enable shortcuts (Ctrl+S/Cmd+S to save) (default is false)
    function playground(opts) {
        var code = $(opts.codeEl);
        var transport = opts['transport'] || new HTTPTransport();
        var running;

        // autoindent helpers.
        function insertTabs(n) {
            // find the selection start and end
            var start = code[0].selectionStart;
            var end     = code[0].selectionEnd;
            // split the textarea content into two, and insert n tabs
            var v = code[0].value;
            var u = v.substr(0, start);
            for (var i=0; i<n; i++) {
                u += "\t";
            }
            u += v.substr(end);
            // set revised content
            code[0].value = u;
            // reset caret position after inserted tabs
            code[0].selectionStart = start+n;
            code[0].selectionEnd = start+n;
        }
        function autoindent(el) {
            var curpos = el.selectionStart;
            var tabs = 0;
            while (curpos > 0) {
                curpos--;
                if (el.value[curpos] == "\t") {
                    tabs++;
                } else if (tabs > 0 || el.value[curpos] == "\n") {
                    break;
                }
            }
            setTimeout(function() {
                insertTabs(tabs);
            }, 1);
        }

        // NOTE(cbro): e is a jQuery event, not a DOM event.
        function handleSaveShortcut(e) {
            if (e.isDefaultPrevented()) return false;
            if (!e.metaKey && !e.ctrlKey) return false;
            if (e.key != "S" && e.key != "s") return false;

            e.preventDefault();

            // Share and save
            share(function(url) {
                window.location.href = url + ".go?download=true";
            });

            return true;
        }

        function keyHandler(e) {
            if (opts.enableShortcuts && handleSaveShortcut(e)) return;

            if (e.keyCode == 9 && !e.ctrlKey) { // tab (but not ctrl-tab)
                insertTabs(1);
                e.preventDefault();
                return false;
            }
            if (e.keyCode == 13) { // enter
                if (e.shiftKey) { // +shift
                    run();
                    e.preventDefault();
                    return false;
                } if (e.ctrlKey) { // +control
                    fmt();
                    e.preventDefault();
                } else {
                    autoindent(e.target);
                }
            }
            return true;
        }
        code.unbind('keydown').bind('keydown', keyHandler);
        var outdiv = $(opts.outputEl).empty();
        var output = $('<pre/>').appendTo(outdiv);

        function body() {
            return $(opts.codeEl).val();
        }
        function setBody(text) {
            $(opts.codeEl).val(text);
        }
        function origin(href) {
            return (""+href).split("/").slice(0, 3).join("/");
        }

        var pushedEmpty = (window.location.pathname == "/");
        function inputChanged() {
            if (pushedEmpty) {
                return;
            }
            pushedEmpty = true;
            $(opts.shareURLEl).hide();
            window.history.pushState(null, "", "/");
        }
        function popState(e) {
            if (e === null) {
                return;
            }
            if (e && e.state && e.state.code) {
                setBody(e.state.code);
            }
        }
        var rewriteHistory = false;
        if (window.history && window.history.pushState && window.addEventListener && opts.enableHistory) {
            rewriteHistory = true;
            code[0].addEventListener('input', inputChanged);
            window.addEventListener('popstate', popState);
        }

        function setError(error) {
            if (running) running.Kill();
            lineClear();
            lineHighlight(error);
            output.empty().addClass("error").text(error);
        }
        function loading() {
            lineClear();
            if (running) running.Kill();
            output.removeClass("error").text('Waiting for remote server...');
        }
        function run() {
            loading();
            fileOut = undefined
            if (typeof opts.jsEl !== 'undefined') fileOut = $(opts.jsEl)
            running = transport.Run(body(), highlightOutput(PlaygroundOutput(output[0], fileOut)), opts);
        }

        function fmt() {
            loading();
            var data = {"body": body()};
            if ($(opts.fmtImportEl).is(":checked")) {
                data["imports"] = true;
            }
            $.ajax(opts.API + "/fmt", {
                data: JSON.stringify(data),
                type: "POST",
                dataType: "json",
                contentType: 'application/json',
                success: function(data) {
                    if (data.Error) {
                        setError(data.Error);
                    } else {
                        setBody(data.Body);
                        setError("");
                    }
                }
            });
        }

        var shareURL; // jQuery element to show the shared URL.
        var sharing = false; // true if there is a pending request.
        var shareCallbacks = [];
        function share(opt_callback) {
            if (opt_callback) shareCallbacks.push(opt_callback);

            if (sharing) return;
            sharing = true;

            var sharingData = body();
            $.ajax(opts.API + "/share", {
                processData: false,
                data: sharingData,
                type: "POST",
                complete: function(xhr) {
                    sharing = false;
                    if (xhr.status != 200) {
                        alert("Server error; try again.");
                        return;
                    }
                    if (opts.shareRedirect) {
                        window.location = opts.shareRedirect + xhr.responseText;
                    }
                    var path = "/p/" + xhr.responseText;
                    var url = origin(window.location) + path;

                    for (var i = 0; i < shareCallbacks.length; i++) {
                        shareCallbacks[i](url);
                    }
                    shareCallbacks = [];

                    if (shareURL) {
                        shareURL.show().val(url).focus().select();

                        if (rewriteHistory) {
                            var historyData = {"code": sharingData};
                            window.history.pushState(historyData, "", path);
                            pushedEmpty = false;
                        }
                    }
                }
            });
        }

        $(opts.runEl).click(run);
        $(opts.fmtEl).click(fmt);

        if (opts.shareEl !== null && (opts.shareURLEl !== null || opts.shareRedirect !== null)) {
            if (opts.shareURLEl) {
                shareURL = $(opts.shareURLEl).hide();
            }
            $(opts.shareEl).click(function() {
                share();
            });
        }

        if (opts.toysEl !== null) {
            $(opts.toysEl).bind('change', function() {
                var toy = $(this).val();
                $.ajax(opts.API + "/doc/play/"+toy, {
                    processData: false,
                    type: "GET",
                    complete: function(xhr) {
                        if (xhr.status != 200) {
                            alert("Server error; try again.");
                            return;
                        }
                        setBody(xhr.responseText);
                    }
                });
            });
        }
    }

    window.playground = playground;
})();

// formatGridColumns returns the expected format for grid-template-columns
function formatGridColumns(widths) {
    return widths[0] + "px 0.4em " + widths[1] + "px 50px"
}

// containParentClass will transcend up from an element and check for specific class name until it hits document
function containParentClass(target, className, parent) {
    if (target == null) return
    if (target == document) return
    if (elementHasClassName(target, className)) return target
    if (target == parent) return

    return containParentClass(target.parentElement, className, parent)
}

// elementHasClassName checks if a target has a specific class name
function elementHasClassName(target, className) {
    if (target.classList == undefined) return false
    if (!target.classList.contains(className)) return false
    return true
}

// Editor is the constructor for the whole Editor process, opening, closing, updating etc.
var Editor = function(config) {
    var self = this
    var offset = 0;
    var isDragging = false;
    var lastPos = null;
    var lastWidth = [];

    this.panels = config.panels
    this.code = config.code
    this.output = config.output
    this.console = config.console
    this.preview = config.preview
    this.api = config.api

    // Add a tooltip to all of the panels
    for (var panel of this.panels) {
        new Tooltip(document.getElementById("grid").getElementsByClassName("sidebar")[0].getElementsByClassName(panel.name)[0], panel.tooltip)
    }

    // Look for click events on the panels
    document.addEventListener("click", function(e) {
        // Check if the click event is a panel
        var target = containParentClass(e.target, "sidebar-box")
        if (target == undefined) return

        // get the panel name
        var name = ""
        for (var clazz of target.classList) {
            if (clazz == "sidebar-box") continue
            name = clazz
            break
        }

        // if no panel name was found, then simply close everything
        // could happen if you want a button for closing all panels and such
        if (name == "") {
            self.updateSidebar("")
            return
        }

        // update the panel/sidebar
        for (var panel of self.panels) {
            if (panel.name == name) {
                self.updateSidebar(panel)
                return
            }
        }

        self.updateSidebar("")
        return
    })

    // detect if someone is trying to move the resizeableVertical
    document.getElementById("resizableVertical").addEventListener("mousedown", function(e) {
        e.preventDefault()
        isDragging = true;
        lastPos = e.clientX

        columns = window.getComputedStyle(document.getElementById("grid")).gridTemplateColumns.split(" ")
        lastWidth = [parseFloat(columns[0]), parseFloat(columns[2])]
    })

    document.getElementById("resizableVertical").addEventListener("mouseup", function() {
        isDragging = false;
        offset = 0
    })
    
    document.getElementsByTagName("body")[0].addEventListener("mousemove", function(e) {
        // check if someone is currently dragging the element
        if (!isDragging) return
        if (e.clientX == lastPos) return

        // calculate by how many pixels the element has moved
        offset = offset - (e.clientX - lastPos)
        lastPos = e.clientX
        
        // check if a panel is actually opened
        var panelopened = false
        var divs = document.getElementById("grid").getElementsByClassName("panel")[0].children
        for (var v of divs) {
            if (v.style.display != "none") {
                panelopened = true
                break
            }
        }

        // update the gridTemplateColumns with the new offset
        if (panelopened) {
            document.getElementById("grid").style.gridTemplateColumns = formatGridColumns([lastWidth[0] - offset, lastWidth[1] + offset])
            var panel = self.opened()
            if (typeof panel != "undefined") self.refresh(panel)
        }
    })

    document.getElementById("run").addEventListener("click", function(e) {
        var body = self.code.value
        fetch(self.api + "/compile", {
            body: JSON.stringify({"version": 2, "body": body}),
            method: "POST",
            headers: {
                "content-type": "application/json"
            }
        })
            .then(response => response.json())
            .then(response => {
                
            })
    })
}

// updateSidebar takes care of closing and opening all of the sidebars
Editor.prototype.updateSidebar = function(panel) {
    // check if there is a sidebars to open/close or if we should simply close all of them
    if (panel == "") {
        var divs = document.getElementById("grid").getElementsByClassName("panel")[0].children
        for (var v of divs) {
            v.style.display = "none"
        }
        document.getElementById("grid").style.gridTemplateColumns = "auto 0 0 50px"
        return
    }

    // get the current display status and then close all sidebars
    var display = document.getElementById(panel.name).style.display
    var divs = document.getElementById("grid").getElementsByClassName("panel")[0].children
    for (var v of divs) {
        v.style.display = "none"
    }

    // Open/close the sidebar
    // Open
    if (display == "none") {
        if (typeof panel.lastWidth == "undefined") {
            document.getElementById("grid").style.gridTemplateColumns = "auto 0.4em 30em 50px"
        } else {
            document.getElementById("grid").style.gridTemplateColumns = formatGridColumns(panel.lastWidth)
        }
        document.getElementById(panel.name).style.display = ""
        document.getElementById("resizableVertical").style.display = ""
        panel.opened = true

        this.refresh(panel)
    
    // Close
    } else {
        columns = window.getComputedStyle(document.getElementById("grid")).gridTemplateColumns
        lastWidth = [parseFloat(columns[0]), parseFloat(columns[2])]
        panel.lastWidth = lastWidth

        document.getElementById("resizableVertical").style.display = "none"
        document.getElementById("grid").style.gridTemplateColumns = "auto 0 0 50px"
        panel.opened = false
    }
}

// If the sidebar that is opened has an editor, then refresh the editor
Editor.prototype.refresh = function(panel) {
    if (typeof panel.editor != "undefined") {
        panel.editor.refresh()
    }
}

// If a sidebar is opened, return the panel object
Editor.prototype.opened = function() {
    for (var panel of this.panels) {
        if (panel.opened) return panel
    }
}