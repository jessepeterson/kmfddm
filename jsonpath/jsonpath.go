// Package jsonpath is a JSON object "path-based" parser based on github.com/valyala/fastjson.
package jsonpath

import (
	"fmt"
	"strings"
	"sync"

	"github.com/valyala/fastjson"
)

// Handlers process parts of JSON structures at paths.
//
// Paths are dot/period-separated locations in the JSON structure.
// Optionally a Handler can return the slice of "unhandled" paths it may
// have traversed.
// Be judicious with returning errors. They may stop processing of the
// JSON structure before other parts have been reached or other handlers
// run instead returning an error for the entire process.
type Handler interface {
	JSONPath(path string, v *fastjson.Value) ([]string, error)
}

// HandlerFunc is an adapter that allows using functions as Handlers.
type HandlerFunc func(path string, v *fastjson.Value) ([]string, error)

// JSONPath calls f(path, v).
func (f HandlerFunc) JSONPath(path string, v *fastjson.Value) ([]string, error) {
	return f(path, v)
}

// pathHandlerMap maps dot/period-separated path elements to pathHandlers.
type pathHandlerMap map[string]*pathHandlers

// pathHandlers is container for handlers for paths.
type pathHandlers struct {
	handlers     []Handler
	wcHandlers   []Handler
	pathHandlers pathHandlerMap
}

// PathMux traverses JSON objects to handle dot/period-separated paths.
//
// Paths are dot/period-separated locations in the JSON structure.
// TThere are two varieties of handler paths: normal handlers and
// "wildcard" handlers. Normal handlers exactly match the path in the
// JSON and are handled when found. Wildcard handlers, which end in a
// dot/period, are called for every non-JSON object key recursively.
//
// For example the handler path ".foo.bar.baz" is a normal handler and
// would be called if the source JSON looked something like
// `{"foo": {"bar": {"baz": true, "qux": false}}}`. However the handler
// path ".foo.bar." is a wildcard (as denoted by the trailing period)
// and the handler will get called for each of the paths ".foo.bar.baz",
// and ".foo.bar.qux".
//
// Normal and wildcard handlers can be registered "lower" (deeper in the
// JSON structure) than higher-level wildcard handlers. Normal handlers
// will stop traversing the structure to handle any registered handlers.
// Only the "lowest" wildcard handler will be dispatched to if more than
// one match for a path. Multiple handlers (of either type) can be
// registered for a path.
type PathMux struct {
	mu           sync.RWMutex
	pathHandlers pathHandlerMap
}

// NewPathMux returns a new PathMux.
func NewPathMux() *PathMux {
	return new(PathMux)
}

// Handle registers h for path in the JSON structure.
// Multiple handlers can be registered for a path.
// Panic ensues if h is nil.
func (m *PathMux) Handle(path string, h Handler) {
	if h == nil {
		panic("jsonpath: nil handler")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	pathElems := strings.Split(path, ".")

	// the end of the pathElems
	end := len(pathElems) - 1
	// the end of the pathElems for "wildcard" Handlers
	wcEnd := end - 1

	// point at the "root" pathHandler of the muxer
	curPathHandlers := &m.pathHandlers

	for i, pathElem := range pathElems {
		if *curPathHandlers == nil {
			*curPathHandlers = make(pathHandlerMap)
		}
		handlers, ok := (*curPathHandlers)[pathElem]
		if !ok || handlers == nil {
			handlers = &pathHandlers{}
			(*curPathHandlers)[pathElem] = handlers
		}
		if i == wcEnd && pathElems[end] == "" {
			// setup a "wildcard" Handler
			handlers.wcHandlers = append(handlers.wcHandlers, h)
			break
		} else if i == end {
			// setup a Handler
			handlers.handlers = append(handlers.handlers, h)
		}

		// re-point at the next pathHandler
		curPathHandlers = &handlers.pathHandlers
	}
}

// HandleFunc registers f for path in the JSON structure.
func (m *PathMux) HandleFunc(path string, f HandlerFunc) {
	m.Handle(path, f)
}

// processValue recursively traverses JSON structure v.
func processValue(handlerMap pathHandlerMap, pathElems []string, curElem int, v *fastjson.Value, wcHandlers []Handler) ([]string, error) {
	curPath := pathElems[curElem]
	path := strings.Join(pathElems, ".")
	handlers, ok := handlerMap[curPath]
	if ok && handlers != nil && len(handlers.wcHandlers) > 0 {
		// everything from this position and deeper will have wcHandlers
		// (we pass wcHandlers to the recursive call). this faciliates
		// wildcard handlers.
		wcHandlers = handlers.wcHandlers
	}
	var unhandled, tmpUnhandled []string
	var err error
	if (!ok || handlers == nil) && len(wcHandlers) < 1 {
		// end of the line. if we're not in the pathHandlerMap and
		// there are no wildcard handlers at this position then
		// nothing will be found for this path. report it back up the
		// recursive call.
		return []string{path}, nil
	} else if handlers != nil && len(handlers.handlers) > 0 {
		// if we have specific handlers for this path then run them
		// (and only them).
		for i, h := range handlers.handlers {
			tmpUnhandled, err = h.JSONPath(path, v)
			unhandled = append(unhandled, tmpUnhandled...)
			if err != nil {
				return unhandled, fmt.Errorf("Handler %d at path %s: %w", i, path, err)
			}
		}
		return unhandled, nil
	}
	switch v.Type() {
	case fastjson.TypeObject:
		o, err := v.Object()
		if err != nil {
			return unhandled, fmt.Errorf("getting object at path %s: %w", path, err)
		}
		if handlers == nil {
			handlers = &pathHandlers{}
		}
		o.Visit(func(k []byte, v *fastjson.Value) {
			if err != nil {
				// don't go further if we have any errors
				return
			}
			tmpUnhandled, err = processValue(handlers.pathHandlers, append(pathElems, string(k)), curElem+1, v, wcHandlers)
			// accumulate our unhandled paths
			unhandled = append(unhandled, tmpUnhandled...)
		})
		if err != nil {
			return unhandled, fmt.Errorf("visiting object path %s: %w", path, err)
		}
	default:
		if len(wcHandlers) > 0 {
			// dispatch to wildcard handlers
			for i, h := range wcHandlers {
				tmpUnhandled, err = h.JSONPath(path, v)
				unhandled = append(unhandled, tmpUnhandled...)
				if err != nil {
					return unhandled, fmt.Errorf("wildcard Handler %d at path %s: %w", i, path, err)
				}
			}
			return unhandled, nil
		}
	}
	return unhandled, nil
}

// JSONPath parses v dispatching to any registered handlers for paths.
func (m *PathMux) JSONPath(path string, v *fastjson.Value) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return processValue(m.pathHandlers, strings.Split(path, "."), 0, v, nil)
}

// StripAndAddPrefix is a middleware that removes pathPrefix from
// handler calls and adds it back when returning unhandled paths.
func StripAndAddPrefix(pathPrefix string, next Handler) Handler {
	if pathPrefix == "" {
		return next
	}
	return HandlerFunc(func(path string, v *fastjson.Value) ([]string, error) {
		p := strings.TrimPrefix(path, pathPrefix)
		unhandled, err := next.JSONPath(p, v)
		unhandledPrefix := make([]string, len(unhandled))
		for i, v := range unhandled {
			unhandledPrefix[i] = pathPrefix + v
		}
		return unhandledPrefix, err
	})
}

// AddPrefix is a middleware that prefixes pathPrefix to handler calls.
func AddPrefix(pathPrefix string, next Handler) Handler {
	if pathPrefix == "" {
		return next
	}
	return HandlerFunc(func(path string, v *fastjson.Value) ([]string, error) {
		return next.JSONPath(pathPrefix+path, v)
	})
}
