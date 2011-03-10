package javascriptcore_test

import(
	"testing"
	js "javascriptcore"
)

type reflect_object struct {
	I	int
	F	float64
	S	string
}

func TestMakeFunctionWithCallback(t *testing.T) {
	var flag bool
	callback := func (ctx *js.Context, obj *js.Object, thisObject *js.Object, _ []*js.Value ) (*js.Value, *js.Value){
		flag = true
		return nil, nil
	}

	ctx := js.NewContext()
	defer ctx.Release()

	fn := ctx.MakeFunctionWithCallback( callback )
	if fn == nil {
		t.Errorf( "ctx.MakeFunctionWithCallback failed" )
		return
	}
	if !ctx.IsFunction( fn ) {
		t.Errorf( "ctx.MakeFunctionWithCallback returned value that is not a function" )
	}
	if ctx.ToStringOrDie( fn.GetValue() ) != "nativecallback" {
		t.Errorf( "ctx.MakeFunctionWithCallback returned value that does not convert to property string" )
	}
	ctx.CallAsFunction( fn, nil, []*js.Value{} )
	if !flag {
		t.Errorf( "Native function did not execute" )
	}
}

func TestMakeFunctionWithCallback2(t *testing.T) {
	callback := func (ctx *js.Context, obj *js.Object, thisObject *js.Object, args []*js.Value ) (*js.Value, *js.Value){
		if len(args)!=2 {
			return nil, nil
		}

		a := ctx.ToNumberOrDie( args[0] )
		b := ctx.ToNumberOrDie( args[1] )
		return ctx.NewNumberValue( a + b ), nil
	}

	ctx := js.NewContext()
	defer ctx.Release()

	fn := ctx.MakeFunctionWithCallback( callback )
	a := ctx.NewNumberValue( 1.5 )
	b := ctx.NewNumberValue( 3.0 )
	val, err := ctx.CallAsFunction( fn, nil, []*js.Value{ a, b } )
	if err != nil || val == nil {
		t.Errorf( "Error executing native function" )
	}
	if ctx.ToNumberOrDie(val)!=4.5 {
		t.Errorf( "Native function did not return the correct value" )
	}
}

func TestMakeNativeObject(t *testing.T) {
	obj := &reflect_object{ 2, 3.0, "four" }

	ctx := js.NewContext()
	defer ctx.Release()

	v := ctx.MakeNativeObject( obj )
	ctx.ObjectSetProperty( ctx.GlobalObject(), "n", v.GetValue(), 0 )

	// Following script access should be successful
	ret, err := ctx.EvaluateScript( "n.F", nil, "./testing.go", 1 )
	if err != nil || ret == nil {
		t.Errorf( "ctx.EvaluateScript returned an error (or did not return a result)" )
		return
	}
	if !ctx.IsNumber( ret ) {
		t.Errorf( "ctx.EvaluateScript did not return 'number' result when accessing native object's non-existent field." )
	}
	num := ctx.ToNumberOrDie( ret )
	if num != 3.0 {
		t.Errorf( "ctx.EvaluateScript incorrect value when accessing native object's field." )
	}

	// following script access should fail
	ret, err = ctx.EvaluateScript( "n.noexist", nil, "./testing.go", 1 )
	if err != nil || ret == nil {
		t.Errorf( "ctx.EvaluateScript returned an error (or did not return a result)" )
	}
	if !ctx.IsUndefined( ret ) {
		t.Errorf( "ctx.EvaluateScript did not return 'undefined' result when accessing native object's non-existent field." )
	}

	// following script access should succeed
	ret, err = ctx.EvaluateScript( "n.S", nil, "./testing.go", 1 )
	if err != nil || ret == nil {
		t.Errorf( "ctx.EvaluateScript returned an error (or did not return a result)" )
	}
	if !ctx.IsString( ret ) {
		t.Errorf( "ctx.EvaluateScript did not return 'string' result when accessing native object's non-existent field." )
	}
	str := ctx.ToStringOrDie( ret )
	if str != "four" {
		t.Errorf( "ctx.EvaluateScript incorrect value when accessing native object's field." )
	}
}

func TestMakeRegExp(t *testing.T) {
	tests := []string{ "\\bt[a-z]+\\b", "[0-9]+(\\.[0-9]*)?" }

	ctx := js.NewContext()
	defer ctx.Release()

	for _, item := range tests {
		r, err := ctx.MakeRegExp( item )
		if err != nil {
			t.Errorf( "ctx.MakeRegExp failed on string %v with error %v", item, err )
		}
		if ctx.ToStringOrDie( r.GetValue() ) != "/" + item + "/" {
			t.Errorf( "Error compling regexp %s", item )
		}
	}
}

func TestMakeRegExpFromValues(t *testing.T) {
	tests := []string{ "\\bt[a-z]+\\b", "[0-9]+(\\.[0-9]*)?" }

	ctx := js.NewContext()
	defer ctx.Release()

	for _, item := range tests {
		params := []*js.Value{ ctx.NewStringValue( item ) }
		r, err := ctx.MakeRegExpFromValues( params )
		if err != nil {
			t.Errorf( "ctx.MakeRegExp failed on string %v with error %v", item, err )
		}
		if ctx.ToStringOrDie( r.GetValue() ) != "/" + item + "/" {
			t.Errorf( "Error compling regexp %s", item )
		}
	}
}

func TestMakeFunction(t *testing.T) {
	ctx := js.NewContext()
	defer ctx.Release()

	fn, err := ctx.MakeFunction( "myfun", []string{ "a", "b" }, "return a+b;", "./testing.go", 1 )
	if err != nil {
		t.Errorf( "ctx.MakeFunction failed with %v", err )
	}
	if !ctx.IsFunction( fn ) {
		t.Errorf( "ctx.MakeFunction did not return a function object" )
	}
}

func TestMakeCallAsFunction(t *testing.T) {
	ctx := js.NewContext()
	defer ctx.Release()

	fn, err := ctx.MakeFunction( "myfun", []string{ "a", "b" }, "return a+b;", "./testing.go", 1 )
	if err != nil {
		t.Errorf( "ctx.MakeFunction failed with %v", err )
	}
	
	a := ctx.NewNumberValue( 1.5 )
	b := ctx.NewNumberValue( 3.0 )
	val, err := ctx.CallAsFunction( fn, nil, []*js.Value{ a, b } )
	if err != nil {
		t.Errorf( "ctx.CallAsFunction failed with %v", err )
	}
	if !ctx.IsNumber( val ) {
		t.Errorf( "ctx.CallAsFunction did not compute the right value" )
	}

	num := ctx.ToNumberOrDie( val )
	if num != 4.5 {
		t.Errorf( "ctx.CallAsFunction did not compute the right value" )
	}
}

