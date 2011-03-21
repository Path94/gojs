package javascriptcore

// #include <stdlib.h>
// #include <JavaScriptCore/JSStringRef.h>
// #include <JavaScriptCore/JSObjectRef.h>
// #include "callback.h"
import "C"
import "os"
import "reflect"
import "runtime"
import "unsafe"

var (
	nativecallback C.JSClassRef
	nativecallback_typ interface{}
	nativefunction C.JSClassRef
	nativeobject C.JSClassRef
)

type Stringer interface {
	String() string
}

func init() {
	// Create the class definition for JavaScriptCore
	nativecallback = C.JSClassDefinition_NativeCallback()
	if nativecallback == nil {
		panic( os.ENOMEM )
	}

	// Get the Go type information to recreate the callback
	var dummy GoFunctionCallback
	nativecallback_typ, _ = unsafe.Reflect( dummy )

	// Create the class definition for JavaScriptCore
	nativeobject = C.JSClassDefinition_NativeObject()
	if nativeobject == nil {
		panic( os.ENOMEM )
	}

	// Create the class definition for JavaScriptCore
	nativefunction = C.JSClassDefinition_NativeFunction()
	if nativefunction == nil {
		panic( os.ENOMEM )
	}
}

func value_to_javascript( ctx *Context, value reflect.Value ) *Value {
	switch value.(type) {
		case (*reflect.IntValue):
			r := value.(*reflect.IntValue).Get()
			return ctx.NewNumberValue( float64(r) )
		case (*reflect.FloatValue):
			r := value.(*reflect.FloatValue).Get()
			return ctx.NewNumberValue( r )
		case (*reflect.StringValue):
			r := value.(*reflect.StringValue).Get()
			return ctx.NewStringValue( r )
	}

	return nil
}

func recover_to_javascript( ctx *Context, r interface{} ) *Value {
	if re, ok := r.(os.Error); ok {
		// TODO:  Check for error return from NewError
		ret, _ := ctx.NewError( re.String() )		
		return (*Value)(unsafe.Pointer(ret))
	}
	if str, ok := r.(Stringer); ok {
		ret, _ := ctx.NewError( str.String() )		
		return (*Value)(unsafe.Pointer(ret))
	}
	if str, ok := reflect.NewValue(r).(*reflect.StringValue); ok {
		ret, _ := ctx.NewError( str.Get() )		
		return (*Value)(unsafe.Pointer(ret))
	}

	// TODO:  Check for error return from NewError
	ret, _ := ctx.NewError( "Internal Go error" )		
	return (*Value)(unsafe.Pointer(ret))
}

func javascript_to_reflect( ctx *Context, param []*Value ) []reflect.Value {
	ret := make( []reflect.Value, len(param) )

	for index, item := range param {
		var goval interface{}

		switch ctx.ValueType( item ) {
		case TypeBoolean:
			goval = ctx.ToBoolean( item )
		case TypeNumber:
			goval = ctx.ToNumberOrDie( item )
		case TypeString:
			goval = ctx.ToStringOrDie( item )
		default:
			panic( "Parameter can not be converted to Go native type." )
		}

		ret[index] = reflect.NewValue( goval )
	}

	return ret
}

func javascript_to_value( field reflect.Value, ctx *Context, value *Value, exception *unsafe.Pointer )  {
	switch field.(type) {
		case *reflect.StringValue:
			str, err := ctx.ToString( value )
			if err == nil {
				field.(*reflect.StringValue).Set( str )
			} else {
				*exception = unsafe.Pointer( err )
			}

		case *reflect.FloatValue:
			flt, err := ctx.ToNumber( value )
			if err == nil {
				field.(*reflect.FloatValue).Set( flt )
			} else {
				*exception = unsafe.Pointer( err )
			}

		case *reflect.IntValue:
			flt, err := ctx.ToNumber( value )
			if err == nil {
				field.(*reflect.IntValue).Set( int64(flt) )
			} else {
				*exception = unsafe.Pointer( err )
			}

		default:
			panic( "Parameter can not be converted to Go native type." )
	}
}

//=========================================================
// Native Callback
//---------------------------------------------------------

type GoFunctionCallback func(ctx *Context, obj *Object, thisObject *Object, arguments []*Value) (ret *Value)

func (ctx *Context) NewFunctionWithCallback( callback GoFunctionCallback ) *Object {
	_, addr := unsafe.Reflect( callback )

	ret := C.JSObjectMake( C.JSContextRef(unsafe.Pointer(ctx)), nativecallback, addr )
	return (*Object)(unsafe.Pointer(ret))
}

//export nativecallback_CallAsFunction_go
func nativecallback_CallAsFunction_go( data unsafe.Pointer, ctx unsafe.Pointer, obj unsafe.Pointer, thisObject unsafe.Pointer, argumentCount uint, arguments unsafe.Pointer, exception *unsafe.Pointer ) unsafe.Pointer {
	defer func() {
		if r := recover(); r != nil {
			*exception = unsafe.Pointer( recover_to_javascript( (*Context)(ctx), r ) )
		}
	}()

	ret := unsafe.Unreflect( nativecallback_typ, data ).(GoFunctionCallback)(
		(*Context)(ctx), (*Object)(obj), (*Object)(thisObject), (*[1<<14]*Value)(arguments)[0:argumentCount] )
	return unsafe.Pointer(ret)
}

//=========================================================
// Native Function
//---------------------------------------------------------

func (ctx *Context) NewFunctionWithNative( fn interface{} ) *Object {
	// Sanity checks on the function
	if typ := reflect.Typeof( fn ).(*reflect.FuncType); typ.NumOut() > 1 {
		panic( "Bad native function:  too many output parameters" )
	}

	typ, addr := unsafe.Reflect( fn )
	typptr := typ.(*runtime.FuncType)
	data := C.new_nativeobject_data( unsafe.Pointer(typptr), addr )

	ret := C.JSObjectMake( C.JSContextRef(unsafe.Pointer(ctx)), nativefunction, data )
	return (*Object)(unsafe.Pointer(ret))
}



func docall( ctx *Context, val *reflect.FuncValue, in []reflect.Value ) (*Value) {
	out := val.Call( in )
	if len(out) == 0 {
		return nil
	}
	// len(out) should be equal to 1
	ret := value_to_javascript( ctx, out[0] )
	if ret == nil {
		panic( "Internal GO error" )
	}
	return ret
}

//export nativefunction_CallAsFunction_go
func nativefunction_CallAsFunction_go( data unsafe.Pointer, ctx unsafe.Pointer, _ unsafe.Pointer, _ unsafe.Pointer, argumentCount uint, arguments unsafe.Pointer, exception *unsafe.Pointer ) unsafe.Pointer {
	defer func() {
		if r := recover(); r != nil {
			*exception = unsafe.Pointer( recover_to_javascript( (*Context)(ctx), r ) )
		}
	}()

	// recover the object
	obji := unsafe.Unreflect( (*C.nativeobject_data)(data).typ, (*C.nativeobject_data)(data).addr )
	typ := reflect.Typeof( obji ).(*reflect.FuncType)
	val := reflect.NewValue( obji ).(*reflect.FuncValue)

	// Do the number of input parameters match?
	if typ.NumIn() != int(argumentCount) {
		panic( "Incorrect number of function arguments" )
	}

	// Perform the call
	if typ.NumIn() == 0 {
		ret := docall( (*Context)(ctx), val, nil )
		return unsafe.Pointer(ret)
	}

	param := javascript_to_reflect( (*Context)(ctx), (*[1<<14]*Value)(arguments)[0:argumentCount] )
	ret := docall( (*Context)(ctx), val, param )
	return unsafe.Pointer( ret )
}

//=========================================================
// Native Object
//---------------------------------------------------------

func (ctx *Context) NewNativeObject( obj interface{} ) *Object {
	// The obj must be a pointer to a struct
	// TODO:  add error checking code

	typ, addr := unsafe.Reflect( obj )
	typptr := typ.(*runtime.PtrType)
	data := C.new_nativeobject_data( unsafe.Pointer(typptr), addr )

	ret := C.JSObjectMake( C.JSContextRef(unsafe.Pointer(ctx)), nativeobject, data )
	return (*Object)(unsafe.Pointer(ret))
}

//export nativeobject_GetProperty_go
func nativeobject_GetProperty_go( data, ctx, _, propertyName unsafe.Pointer, exception *unsafe.Pointer ) unsafe.Pointer {
	// Get name of property as a go string
	name := (*String)(propertyName).String()

	// Reconstruct the object interface
	obji := unsafe.Unreflect( (*C.nativeobject_data)(data).typ, (*C.nativeobject_data)(data).addr )

	// Drill down through reflect to find the property
	objv := reflect.NewValue( obji )
	if ptrvalue, ok := objv.(*reflect.PtrValue); ok {
		objv = ptrvalue.Elem()
	}
	strvalue, ok := objv.(*reflect.StructValue)
	if !ok {
		return nil
	}

	field := strvalue.FieldByName( name )
	if field == nil {
		return nil
	}

	return unsafe.Pointer( value_to_javascript( (*Context)(ctx), field ) )
}

func internal_go_error( ctx *Context ) *Value {
	param := ctx.NewStringValue( "Internal Go error." )
	
	exception := (*Value)(nil)
	ret := C.JSObjectMakeError( C.JSContextRef(unsafe.Pointer(ctx)), 
		C.size_t(1), (*C.JSValueRef)( unsafe.Pointer( &param ) ),
		(*C.JSValueRef)(unsafe.Pointer(&exception)) )
	if ret != nil {
		return (*Value)(unsafe.Pointer(ret))
	}
	if exception != nil{
		return exception
	}
	panic( "Internal Go error." )
}

//export nativeobject_SetProperty_go
func nativeobject_SetProperty_go( data, ctx, _, propertyName, value unsafe.Pointer, exception *unsafe.Pointer ) C.char {
	// Get name of property as a go string
	name := (*String)(propertyName).String()

	// Reconstruct the object interface
	obji := unsafe.Unreflect( (*C.nativeobject_data)(data).typ, (*C.nativeobject_data)(data).addr )

	// Drill down through reflect to find the property
	objv := reflect.NewValue( obji )
	if ptrvalue, ok := objv.(*reflect.PtrValue); ok {
		objv = ptrvalue.Elem()
	}
	strvalue, ok := objv.(*reflect.StructValue)
	if !ok {
		*exception = unsafe.Pointer( internal_go_error( (*Context)(ctx) ) )
		return 1
	}

	field := strvalue.FieldByName( name )
	if field == nil {
		return 0
	}

	javascript_to_value( field, (*Context)(ctx), (*Value)(value), exception )
	return 1
}

//export nativeobject_ConvertToString_go
func nativeobject_ConvertToString_go( data, ctx, obj unsafe.Pointer ) unsafe.Pointer {
	// Reconstruct the object interface
	obji := unsafe.Unreflect( (*C.nativeobject_data)(data).typ, (*C.nativeobject_data)(data).addr )

	// Can we get a string?
	if stringer, ok := obji.(Stringer); ok {
		str := stringer.String()
		ret := NewString( str )
		return unsafe.Pointer( ret )
	}

	return nil
}

