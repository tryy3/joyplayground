// formatGridColumns returns the expected format for grid-template-columns
function formatGridColumns(widths) {
    return widths[0] + "px 0.4em " + widths[1] + "px 50px"
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
        for (var path of e.path) {
            if (path.classList == undefined) continue
            if (path.classList.contains("sidebar-box")) {
                // get the panel name
                var name = ""
                for (var clazz of path.classList) {
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
            }
        }
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
        var body = self.code.getValue()
        fetch(self.api + "/compile", {
            body: JSON.stringify({"version": 2, "body": body}),
            method: "POST",
            headers: {
                "content-type": "application/json"
            },
            mode: 'cors'
        })
        .then(response => response.json())
        .then(response => {
            if (response.error != "") {
                self.output.setValue(response.error)
            } else {
                self.output.setValue(response.compiled)
            }
        })
    })

    document.getElementById("fmt").addEventListener("click", function(e) {
        var body = self.code.getValue()
        var imports = document.getElementById("importsBox").querySelector("input[type=checkbox]").checked
        fetch(self.api+"/fmt", {
            body: JSON.stringify({body,imports}),
            method: "POST",
            headers: {
                "content-type": "application/json"
            },
            mode: 'cors'
        })
        .then(response => response.json())
        .then(response => {
            if (response.Error != "") {
                // TODO: Make this some type of notification
                alert(response.Error)
            } else {
                self.code.setValue(response.Body)
            }
        })
    })

    document.getElementById("share").addEventListener("click", function(e) {
        var body = self.code.getValue()
        fetch(self.api+"/share", {
            body: body,
            method: "POST",
            headers: {
                "content-type": "application/json"
            },
            mode: 'cors'
        })
        .then(response => response.text())
        .then(response => {
            var path = "/p/" + response;
            var url = (""+window.location).split("/").slice(0, 3).join("/") + path
            var embed = document.getElementById("embedBox").querySelector("input[type=checkbox]").checked
            if (embed) {
                url = `<iframe src="${response}" framebordser="0" style="width:100%; height:100%;"><a href="${response}">see this code in joyplayground.tryy3.us</a></iframe>`;
            }

            document.getElementById("shareURL").style.display=""
            document.getElementById("shareURL").value=url

            var historyData = {"code": body}
            window.history.pushState(historyData, "", path)

            setTimeout(function() {
                document.getElementById("shareURL").focus()
                document.getElementById("shareURL").select()
            }, 100)
        })
    })

    if (window.location.pathname.indexOf("/p/") !== -1) {
        fetch(self.api+window.location.pathname, {
            method: "GET",
            headers: {
                "content-type": "application/json"
            },
            mode: 'cors'
        })
        .then(response => response.text())
        .then(response => {
            self.code.setValue(response)
        })
    }
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