function bubbleIframeMouseMove(iframe) {
    iframe.contentWindow.addEventListener("mousemove", function(e) {
        var boundingClientRect = iframe.getBoundingClientRect();

        var evt = new CustomEvent("mousemove", {bubbles: true, cancelable: false})
        evt.clientX = e.clientX + boundingClientRect.left;
        evt.clientY = e.clientY + boundingClientRect.top;

        iframe.dispatchEvent(evt);
    })
}

// formatGridColumns returns the expected format for grid-template-columns
function formatGridColumns(widths) {
    return widths[0] + "px 1vw " + widths[1] + "px 5vw"
}

// containParentClass will transcend up from an element and check for specific class name until it finds className
function containParentClass(target, className, parent) {
    if (target == null) return
    if (typeof target == "undefined") return
    if (target == parent) return
    if (elementHasClassName(target, className)) return target

    return containParentClass(target.parentElement, className, parent)
}

// elementHasClassName checks if a target has a specific class name
function elementHasClassName(target, className) {
    if (typeof target.classList == "undefined") return false
    if (!target.classList.contains(className)) return false
    return true
}

// loading handles the process of enabling/disabling the loading animation
function loading(element, toggle) {
    // Start loading animations
    if (toggle) {
        for (var el of document.getElementById("controls").getElementsByTagName("button")) {
            el.disabled = true
        }
        element.querySelector("svg").style.display=""
        element.style.cursor="wait"
    } else {
        for (var el of document.getElementById("controls").getElementsByTagName("button")) {
            el.disabled = false
        }
        element.querySelector("svg").style.display="none"
        element.style.cursor="pointer"
    }
}

// Editor is the constructor for the whole Editor process, opening, closing, updating etc.
var Editor = function(config) {
    var self = this
    var offset = 0;
    var isDragging = false;
    var lastPos = null;
    var lastWidth = [];

    this.consoleLogs = [];

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

    bubbleIframeMouseMove(this.getPanelName("live").element.querySelector("iframe"))

    // Look for click events on the panels
    document.addEventListener("click", function(e) {
        // Check if the click event is a panel
        var target = containParentClass(e.target, "sidebar-box")
        if (typeof target == "undefined") return

        // get the panel name
        var name
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
    function downHandler(e) {
        console.log(e)
        e.preventDefault()
        isDragging = true;

        var clientX = e.clientX
        if (e instanceof TouchEvent) {
            clientX = e.touches[0].clientX
        }
        lastPos = clientX

        var columns = window.getComputedStyle(document.getElementById("grid")).gridTemplateColumns.split(" ")
        lastWidth = [parseFloat(columns[0]), parseFloat(columns[2])]
    }

    function upHandler(e) {
        console.log(e)
        e.preventDefault()
        isDragging = false;
        offset = 0
    }

    function moveHandler(e) {
        console.log(e)
        // check if someone is currently dragging the element
        if (!isDragging) return

        var clientX = e.clientX
        if (e instanceof TouchEvent) {
            clientX = e.touches[0].clientX
        }
        if (clientX == lastPos) return
        e.preventDefault()

        // calculate by how many pixels the element has moved
        offset = offset - (clientX - lastPos)
        lastPos = clientX
        
        // check if a panel is actually opened
        var openedPanel = self.opened()
        if (typeof openedPanel != "undefined") {
            document.getElementById("grid").style.gridTemplateColumns = formatGridColumns([lastWidth[0] - offset, lastWidth[1] + offset])
            openedPanel.refresh()
        }
    }

    document.getElementById("resizableVertical").addEventListener("mousedown", downHandler)
    document.getElementById("resizableVertical").addEventListener("touchstart", downHandler)

    document.getElementById("resizableVertical").addEventListener("mouseup", upHandler)
    document.getElementById("resizableVertical").addEventListener("touchend", upHandler)

    var body =
    document.getElementsByTagName("body")[0].addEventListener("mousemove", moveHandler)
    document.getElementsByTagName("body")[0].addEventListener("touchmove", moveHandler)


    document.getElementById("run").addEventListener("click", function(e) {
        var el = this
        loading(el, true)
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
            self.consoleLogs = [];
            if (response.error != "") {
                self.consoleLogs.push({Kind: "stderr", Body: response.error})
                self.consoleUpdate()
            } else {
                self.output.setValue(response.compiled)
                var live = self.getPanelName("live")
                live.element.querySelector("iframe").src = self.api + "/js/" + response.id
                live.setNotification(1)
            }
            self.getPanelName("compiled").setNotification(1)
            loading(el, false)
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

    window.onmessage = function(e) {
        self.consoleLogs.push({Kind: 'stdout', Body: e.data})
        self.consoleUpdate()
    }

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

    for (var el of document.getElementsByName("theme-radio")) {
        el.addEventListener("click", switchTheme)
    }

    var theme = Cookies.get("theme")
    if (typeof theme != "undefined") {
        for (var el of document.getElementsByName("theme-radio")) {
            if (el.value == theme) {
                el.checked = "checked"
                this.switchTheme(theme)
                break
            }
        }
    }

    function switchTheme() {
        var theme = this.value
        self.switchTheme(theme)

        Cookies.set("theme", theme)
    }
}

// switchTheme will take care of everything involving switching themes
// such as adding class to body, changing theme's on editors etc.
Editor.prototype.switchTheme = function(theme) {
    document.getElementsByTagName("body")[0].className = theme

    var editor = this.code
    var output = this.output

    if (theme == "dark") {
        editor.setOption("theme", "dracula")
        output.setOption("theme", "dracula")
    } else {
        editor.setOption("theme", "default")
        output.setOption("theme", "default")
    }
}

// updateSidebar takes care of closing and opening all of the sidebars
Editor.prototype.updateSidebar = function(panel) {
    // check if there is a sidebars to open/close or if we should simply close all of them
    if (panel == "") {
        for (var panel of this.panels) {
            panel.toggle(false)
        }
        document.getElementById("grid").style.gridTemplateColumns = "auto 0 0 5vw"
        return
    }

    // retrieve a panel if one of the panels is open
    var openedPanel = this.opened()

    // close all of the panels that we aren't editing (in case something messed up)
    for (var p of this.panels) {
        if (p.name != panel.name && (typeof openedPanel == "undefined" || openedPanel.name == panel.name)) p.toggle(false)
    }

    // if there was a panel open previously and it's not the same as "panel" then close it
    if (typeof openedPanel != "undefined" && openedPanel.name != panel.name) openedPanel.toggle(false)

    // toggle the panel
    panel.toggle()
}

// consoleUpdate will take care of outputting all of the messages to the "virtual" console
Editor.prototype.consoleUpdate = function() {
    var console = this.getPanelName("console")
    var element = console.element.querySelector("#output pre")

    element.innerHTML = '';
    for (var msg of this.consoleLogs) {
        var m = msg.Body
        m = m.replace(/&/g, '&amp;');
        m = m.replace(/</g, '&lt;');
        m = m.replace(/>/g, '&gt;');

        var span = document.createElement("span")
        span.className = msg.Kind
        span.innerHTML = m
        element.appendChild(span)
    }

    var span = document.createElement("span")
    span.className = "system"
    span.innerHTML = '\nProgram exited'
    element.appendChild(span)

    console.setNotification(this.consoleLogs.length)
}

// If a sidebar is opened, return the panel object
Editor.prototype.opened = function() {
    for (var panel of this.panels) {
        if (panel.status) return panel
    }
}

// Find a panel based on its name
Editor.prototype.getPanelName = function(name) {
    for (var panel of this.panels) {
        if (panel.name == name) return panel
    }
}

// Panel will create a panel object
function Panel(config) {
    this.status = false
    this.notifications = 0

    this.name = config.name
    this.element = config.element
    if (typeof config.editor != "undefined") this.editor = config.editor

    new Tooltip(document.getElementById("grid").getElementsByClassName("sidebar")[0].getElementsByClassName(this.name)[0], config.tooltip)
}

// Will refresh if the panel has an editor
Panel.prototype.refresh = function() {
    if (typeof this.editor != "undefined") this.editor.refresh()
}

// Will toggle the panel on and off
Panel.prototype.toggle = function(status) {
    if (typeof status != "undefined") {
        if (status == this.status) return
    } else {
        status = !this.status
    }
    if (status) {
        if (typeof this.lastWidth == "undefined") {
            document.getElementById("grid").style.gridTemplateColumns = "auto 1vw 35vw 5vw"
        } else {
            document.getElementById("grid").style.gridTemplateColumns = formatGridColumns(this.lastWidth)
        }
        this.element.style.display = ""
        document.getElementById("resizableVertical").style.display = ""

        document.getElementById("sidebar").getElementsByClassName(this.name)[0].classList.add("opened")

        this.refresh()
        this.setNotification(0)
        // Close
    } else {
        var columns = window.getComputedStyle(document.getElementById("grid")).gridTemplateColumns.split(" ")
        this.lastWidth = [parseFloat(columns[0]), parseFloat(columns[2])]

        this.element.style.display = "none"
        document.getElementById("resizableVertical").style.display = "none"
        document.getElementById("grid").style.gridTemplateColumns = "auto 0 0 5vw"

        document.getElementById("sidebar").getElementsByClassName(this.name)[0].classList.remove("opened")
    }
    this.status = status
}

// If the panel isn't opened, it will increase the notification amount
Panel.prototype.setNotification = function(val) {
    if (this.status) return
    if (typeof val == "undefined")
        this.notifications++
    else
        this.notifications = val
    this.displayNotification()
}

// If the panel has any notifications it will display them
Panel.prototype.displayNotification = function() {
    if (this.notifications < 0) return
    var element = document.getElementById("sidebar").querySelector("." + this.name + " .fa-layers-counter")
    if (this.notifications == 0) {
        element.textContent = ""
        element.style.display = "none"
    } else {
        element.textContent = this.notifications.toString()
        element.style.display = ""
    }
}