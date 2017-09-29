String.prototype.format = function() {
    a = this;
    for (k in arguments) {
        a = a.replace("{" + k + "}", arguments[k])
    }
    return a
}

function objToQuery( obj ) {
  return '?'+Object.keys(obj).reduce(function(a,k){a.push(k+'='+encodeURIComponent(obj[k]));return a},[]).join('&')
}

var pocketStriptID = "pocket-script"

function insertPocketScript() {
    var id = "pocket-script"
    if (!document.getElementById(pocketStriptID) && window.PocketEnabled) {
        console.log("inserting pocket stcript")
        var script = document.createElement("script");
        script.id = pocketStriptID;
        script.src="https://widgets.getpocket.com/v1/j/btn.js?v=1";
        document.body.appendChild(script)
    }
}

function removePockteScript() {
    var script = document.getElementById(pocketStriptID)
    if (script) {
        document.body.removeChild(script);
    }
}

function reloadPocketScript() {
    removePockteScript();
    insertPocketScript();
}

const store = new Vuex.Store({
    state: {
        scrollTop: 0,
        items: [],
        folders: [],
        loadingMutex: 0,
        page: 0,
        pageMutex: 0,
        lastPage: false,
        filters: {
            unread: false,
            folder: false,
        },
        hamburgerExpanded: false,
    },
    mutations: {
        scroll: function(state, pos) {
            state.scrollTop = pos
        },

        items: function(state, items) {
            state.items = items
            state.page = 0
        },

        appendPageItems: function(state, items) {
            if (Array.isArray(items) && items.length > 0) {
                state.items = state.items.concat(items)
            } else {
                state.lastPage = true
            }
        },

        folders: function(state, folders) {
            state.folders = folders
        },

        focusFolder: function(state, folder) {
            state.filters.folder = folder
        },

        startLoading: function(state) {
            state.loadingMutex += 1
        },

        stopLoading: function(state) {
            state.loadingMutex -= 1
        },

        lockPage: function(state) {
            state.pageMutex += 1
            state.page += 1
        },

        unlockPage: function(state) {
            state.pageMutex -= 0
        },

        markAsRead: function(state, index) {
            state.items[index].read = true
        },

        toggleNewFilter: function(state) {
            state.filters.unread = !state.filters.unread
        },

        toggleHamburger: function(state) {
            state.hamburgerExpanded = !state.hamburgerExpanded
        },

        closeHamburger: function(state) {
            state.hamburgerExpanded = false
        },
    }
});

function scrollHandler() {
    if (document.documentElement.clientWidth < 768) {
        let progress = 100 * document.body.scrollTop / (document.body.offsetHeight-window.innerHeight)
        if (progress > 95) {
            loadExtraPage()
        }
        store.commit('scroll', document.body.scrollTop)
    }
}
window.onscroll = scrollHandler;


var reloadFeeds = function() {
    let folder = store.state.filters.folder
    let showOnlyUnread = store.state.filters.unread
    let params = {page: 0}

    if (folder) {
        params.folder = folder
    }

    if (showOnlyUnread) {
        params.unread = true
    }

    let url = "/api/feeds{0}".format(objToQuery(params))

    store.commit("startLoading")
    fetch(url, {
        credentials: "same-origin"
    }).then(function(response) {
        if (response.ok) {
            return response.json()
        } else {
            throw new Error("Erro fetching data")
        }
    }).then(function(json) {
        store.commit("items", json)
        store.commit("stopLoading")
    });
}

var loadExtraPage = function() {
    if (!store.state.lastPage) {
        store.commit("lockPage")

        let folder = store.state.filters.folder
        let showOnlyUnread = store.state.filters.unread
        let page = store.state.page
        let params = {page: page}

        if (folder) {
            params.folder = folder
        }

        if (showOnlyUnread) {
            params.unread = true
        }

        let url = "/api/feeds{0}".format(objToQuery(params))

        fetch(url, {
            credentials: "same-origin"
        }).then(function(response) {
            if (response.ok) {
                return response.json()
            } else {
                throw new Error("Erro fetching data")
            }
        }).then(function(json) {
            store.commit("appendPageItems", json)
            store.commit("unlockPage")
        });
    }
}

var reloadFolders = function() {
    store.commit("startLoading")
    fetch("/api/folders", {
        credentials: "same-origin"
    }).then(function(response) {
        if (response.ok) {
            return response.json()
        } else {
            throw new Error("Erro fetching data")
        }
    }).then(function(json) {
        store.commit("folders", json)
        store.commit("stopLoading")
    });
}

reloadFeeds()
reloadFolders()

var markItemReadInDB = function(uuid, index) {
    fetch("/api/feeds/" + uuid + "/read", {
        method: "POST",
        credentials: "same-origin",
    }).then(function(response) {
        if (response.ok) {
            store.commit("markAsRead", index)
        } else {
            console.log(response)
        }
    })
}

var markFolderReadInDB = function() {
    if (store.state.filters.folder) {
        fetch("/api/folders/" + store.state.filters.folder + "/read", {
            method: "POST",
            credentials: "same-origin",
        }).then(function(response) {
            if (response.ok) {
                reloadFolders()
                reloadFeeds()
            } else {
                console.log(response)
            }
        })
    }
}

Vue.component('folders-component', {
    template: '#folders-template',
    computed: {
        folders: function() {
            return store.state.folders
        },
        defaultActive: function() {
            return !store.state.filters.folder
        },
        expanded: function() {
            return store.state.hamburgerExpanded
        }
    },
    methods: {
        titleize: function(string) {
            return string.charAt(0).toUpperCase() + string.slice(1);
        },
        focus: function(folder) {
            store.commit("focusFolder", folder)
            reloadFeeds()
            reloadFolders()
            store.commit("closeHamburger")
        },
        unfocus: function() {
            store.commit("focusFolder", false)
            reloadFeeds()
            store.commit("closeHamburger")
        },
        isActive: function(folder) {
            return folder == store.state.filters.folder
        },
        expandHamburger: function() {
            store.commit("toggleHamburger")
        },
    },
});


Vue.component('filters-component', {
    template: '#filters-template',
    computed: {
        checked: function() {
            return store.state.filters.unread
        },
        folderSelected: function() {
            return store.state.filters.folder
        },
        expanded: function() {
            return store.state.hamburgerExpanded
        }
    },
    methods: {
        toggleUnread: function(e) {
            store.commit("toggleNewFilter")
            store.commit("toggleHamburger")
            reloadFeeds()
        },
        markFolderAsRead: function(e) {
            markFolderReadInDB()
            store.commit("toggleHamburger")
        }
    },
});

Vue.component('list-component', {
    template: '#list-template',
    updated: reloadPocketScript,
    methods: {
        scrollHandler: function() {
            let el = this.$el
            let progress = 100 * el.scrollTop / (el.scrollHeight-el.clientHeight)
            if (progress > 95) {
                loadExtraPage()
            }
            store.commit('scroll', el.scrollTop)
        },
    },
    computed: {
        items: function() {
            return store.state.items
        },
        loading: function() {
            return store.state.loadingMutex > 0
        },
    },
});

Vue.component('content-component', {
    template: '#content-template',
    props: ['item', 'index'],
    computed: {
        unread: function() {
            return !this.item.read
        },
    },
    created: function() {
        this.$on('scroll', this.markAsRead)

        let dis = this
        store.watch(
            function() {
                return store.state.scrollTop
            },
            function(n) {
                dis.$emit('scroll')
            },
        )
    },
    methods: {
        markAsRead: function() {
            if (!this.item.read && (store.state.scrollTop > this.$el.offsetTop)) {
                markItemReadInDB(this.item.uuid, this.index)
            }
        },
    },
});

Vue.component('pocket-button-component', {
    template: '#pocket-button',
    props: ['url'],
});

var vm = new Vue({
    el: '#app'
})
