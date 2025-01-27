package webkit

/*
#cgo linux openbsd freebsd CXXFLAGS: -DWEBVIEW_GTK -std=c++11
#cgo linux openbsd freebsd pkg-config: gtk+-3.0 webkit2gtk-4.1

#define GO_WEBKIT_HEADER
#include "go_webkit.h"

#include <stdlib.h>
#include <stdint.h>

extern void _goWebkitDispatchGoCallback(void *);
static inline void _go_webkit_dispatch_cb(go_webkit_t w, void *arg) {
	_goWebkitDispatchGoCallback(arg);
}
static inline void CgoWebkitDispatch(go_webkit_t w, uintptr_t arg) {
	go_webkit_dispatch(w, _go_webkit_dispatch_cb, (void *)arg);
}

struct binding_context {
	go_webkit_t w;
	uintptr_t index;
};
extern void _goWebkitBindingGoCallback(go_webkit_t, char *, char *, uintptr_t);
static inline void _go_webkit_binding_cb(const char *id, const char *req, void *arg) {
	struct binding_context *ctx = (struct binding_context *) arg;
	_goWebkitBindingGoCallback(ctx->w, (char *)id, (char *)req, ctx->index);
}
static inline void CgoWebkitBind(go_webkit_t w, const char *name, uintptr_t index) {
	struct binding_context *ctx = calloc(1, sizeof(struct binding_context));
	ctx->w = w;
	ctx->index = index;
	go_webkit_bind(w, name, _go_webkit_binding_cb, (void *)ctx);
}
*/
import "C"
import (
	"encoding/json"
	"errors"
	"reflect"
	"runtime"
	"sync"
	"unsafe"
)

func init() {
	// Ensure that main.main is called from the main thread
	runtime.LockOSThread()
}

// Hints are used to configure window sizing and resizing
type Hint int

const (
	// Width and height are default size
	HintNone = C.GO_WEBKIT_HINT_NONE

	// Window size can not be changed by a user
	HintFixed = C.GO_WEBKIT_HINT_FIXED

	// Width and height are minimum bounds
	HintMin = C.GO_WEBKIT_HINT_MIN

	// Width and height are maximum bounds
	HintMax = C.GO_WEBKIT_HINT_MAX
)

type Webkit interface {

	// Run runs the main loop until it's terminated. After this function exits -
	// you must destroy the goWebkit.
	Run()

	// Terminate stops the main loop. It is safe to call this function from
	// a background thread.
	Terminate()

	// Dispatch posts a function to be executed on the main thread. You normally
	// do not need to call this function, unless you want to tweak the native
	// window.
	Dispatch(f func())

	// Destroy destroys a goWebkit and closes the native window.
	Destroy()

	// Window returns a native window handle pointer. When using GTK backend the
	// pointer is GtkWindow pointer, when using Cocoa backend the pointer is
	// NSWindow pointer, when using Win32 backend the pointer is HWND pointer.
	Window() unsafe.Pointer

	// SetTitle updates the title of the native window. Must be called from the UI
	// thread.
	SetTitle(title string)

	// SetSize updates native window size. See Hint constants.
	SetSize(w int, h int, hint Hint)

	// Navigate navigates goWebkit to the given URL. URL may be a data URI, i.e.
	// "data:text/text,<html>...</html>". It is often ok not to url-encode it
	// properly, goWebkit will re-encode it for you.
	Navigate(url string)

	// Init injects JavaScript code at the initialization of the new page. Every
	// time the goWebkit will open a the new page - this initialization code will
	// be executed. It is guaranteed that code is executed before window.onload.
	Init(js string)

	// Eval evaluates arbitrary JavaScript code. Evaluation happens asynchronously,
	// also the result of the expression is ignored. Use RPC bindings if you want
	// to receive notifications about the results of the evaluation.
	Eval(js string)

	// Bind binds a callback function so that it will appear under the given name
	// as a global JavaScript function. Internally it uses go_webkit_init().
	// Callback receives a request string and a user-provided argument pointer.
	// Request string is a JSON array of all the arguments passed to the
	// JavaScript function.
	//
	// f must be a function
	// f must return either value and error or just error
	Bind(name string, f interface{}) error
}

type goWebkit struct {
	w C.go_webkit_t
}

var (
	m        sync.Mutex
	index    uintptr
	dispatch = map[uintptr]func(){}
	bindings = map[uintptr]func(id, req string) (interface{}, error){}
)

func boolToInt(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

// New calls NewWindow to create a new window and a new goWebkit instance. If debug
// is non-zero - developer tools will be enabled (if the platform supports them).
func New(debug bool) Webkit { return NewWindow(debug, nil) }

// NewWindow creates a new goWebkit instance. If debug is non-zero - developer
// tools will be enabled (if the platform supports them). Window parameter can be
// a pointer to the native window handle. If it's non-null - then child WebView is
// embedded into the given parent window. Otherwise a new window is created.
// Depending on the platform, a GtkWindow, NSWindow or HWND pointer can be passed
// here.
func NewWindow(debug bool, window unsafe.Pointer) Webkit {
	w := &goWebkit{}
	w.w = C.go_webkit_create(boolToInt(debug), window)
	return w
}

func (w *goWebkit) Destroy() {
	C.go_webkit_destroy(w.w)
}

func (w *goWebkit) Run() {
	C.go_webkit_run(w.w)
}

func (w *goWebkit) Terminate() {
	C.go_webkit_terminate(w.w)
}

func (w *goWebkit) Window() unsafe.Pointer {
	return C.go_webkit_get_window(w.w)
}

func (w *goWebkit) Navigate(url string) {
	s := C.CString(url)
	defer C.free(unsafe.Pointer(s))
	C.go_webkit_navigate(w.w, s)
}

func (w *goWebkit) SetTitle(title string) {
	s := C.CString(title)
	defer C.free(unsafe.Pointer(s))
	C.go_webkit_set_title(w.w, s)
}

func (w *goWebkit) SetSize(width int, height int, hint Hint) {
	C.go_webkit_set_size(w.w, C.int(width), C.int(height), C.int(hint))
}

func (w *goWebkit) Init(js string) {
	s := C.CString(js)
	defer C.free(unsafe.Pointer(s))
	C.go_webkit_init(w.w, s)
}

func (w *goWebkit) Eval(js string) {
	s := C.CString(js)
	defer C.free(unsafe.Pointer(s))
	C.go_webkit_eval(w.w, s)
}

func (w *goWebkit) Dispatch(f func()) {
	m.Lock()
	for ; dispatch[index] != nil; index++ {
	}
	dispatch[index] = f
	m.Unlock()
	C.CgoWebkitDispatch(w.w, C.uintptr_t(index))
}

//export _goWebkitDispatchGoCallback
func _goWebkitDispatchGoCallback(index unsafe.Pointer) {
	m.Lock()
	f := dispatch[uintptr(index)]
	delete(dispatch, uintptr(index))
	m.Unlock()
	f()
}

//export _goWebkitBindingGoCallback
func _goWebkitBindingGoCallback(w C.go_webkit_t, id *C.char, req *C.char, index uintptr) {
	m.Lock()
	f := bindings[uintptr(index)]
	m.Unlock()
	jsString := func(v interface{}) string { b, _ := json.Marshal(v); return string(b) }
	status, result := 0, ""
	if res, err := f(C.GoString(id), C.GoString(req)); err != nil {
		status = -1
		result = jsString(err.Error())
	} else if b, err := json.Marshal(res); err != nil {
		status = -1
		result = jsString(err.Error())
	} else {
		status = 0
		result = string(b)
	}
	s := C.CString(result)
	defer C.free(unsafe.Pointer(s))
	C.go_webkit_return(w, id, C.int(status), s)
}

func (w *goWebkit) Bind(name string, f interface{}) error {
	v := reflect.ValueOf(f)
	// f must be a function
	if v.Kind() != reflect.Func {
		return errors.New("only functions can be bound")
	}
	// f must return either value and error or just error
	if n := v.Type().NumOut(); n > 2 {
		return errors.New("function may only return a value or a value+error")
	}

	binding := func(id, req string) (interface{}, error) {
		raw := []json.RawMessage{}
		if err := json.Unmarshal([]byte(req), &raw); err != nil {
			return nil, err
		}

		isVariadic := v.Type().IsVariadic()
		numIn := v.Type().NumIn()
		if (isVariadic && len(raw) < numIn-1) || (!isVariadic && len(raw) != numIn) {
			return nil, errors.New("function arguments mismatch")
		}
		args := []reflect.Value{}
		for i := range raw {
			var arg reflect.Value
			if isVariadic && i >= numIn-1 {
				arg = reflect.New(v.Type().In(numIn - 1).Elem())
			} else {
				arg = reflect.New(v.Type().In(i))
			}
			if err := json.Unmarshal(raw[i], arg.Interface()); err != nil {
				return nil, err
			}
			args = append(args, arg.Elem())
		}
		errorType := reflect.TypeOf((*error)(nil)).Elem()
		res := v.Call(args)
		switch len(res) {
		case 0:
			// No results from the function, just return nil
			return nil, nil
		case 1:
			// One result may be a value, or an error
			if res[0].Type().Implements(errorType) {
				if res[0].Interface() != nil {
					return nil, res[0].Interface().(error)
				}
				return nil, nil
			}
			return res[0].Interface(), nil
		case 2:
			// Two results: first one is value, second is error
			if !res[1].Type().Implements(errorType) {
				return nil, errors.New("second return value must be an error")
			}
			if res[1].Interface() == nil {
				return res[0].Interface(), nil
			}
			return res[0].Interface(), res[1].Interface().(error)
		default:
			return nil, errors.New("unexpected number of return values")
		}
	}

	m.Lock()
	for ; bindings[index] != nil; index++ {
	}
	bindings[index] = binding
	m.Unlock()
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	C.CgoWebkitBind(w.w, cname, C.uintptr_t(index))
	return nil
}
