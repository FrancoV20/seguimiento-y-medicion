package web

import "net/http"

// MsgOperationFailed es el texto único cuando falla un guardado (HU-20b).
const MsgOperationFailed = "La operación no pudo completarse"

// FormView lleva a la plantilla los valores enviados y el error de cada campo.
type FormView struct {
	Values map[string]string
	Errors map[string]string
}

// NewFormView crea un formulario vacío.
func NewFormView() *FormView {
	return &FormView{Values: map[string]string{}, Errors: map[string]string{}}
}

// FormFromRequest lee del envío los campos indicados, para poder conservarlos.
func FormFromRequest(r *http.Request, names ...string) *FormView {
	f := NewFormView()
	_ = r.ParseForm()
	for _, n := range names {
		f.Values[n] = r.PostFormValue(n)
	}
	return f
}

// AddError asocia un mensaje de error a un campo.
func (f *FormView) AddError(name, msg string) {
	if f.Errors == nil {
		f.Errors = map[string]string{}
	}
	f.Errors[name] = msg
}

// HasErrors indica si algún campo tiene error.
func (f *FormView) HasErrors() bool { return f != nil && len(f.Errors) > 0 }

// FieldView son los datos de un campo para el parcial "field".
type FieldView struct {
	Name, Label, Type, Value, Error string
}

// Field arma el campo name con su valor conservado y su error, si lo hay.
func (f *FormView) Field(name, label, typ string) FieldView {
	fv := FieldView{Name: name, Label: label, Type: typ}
	if f != nil {
		fv.Value = f.Values[name]
		fv.Error = f.Errors[name]
	}
	return fv
}

// RedirectSeeOther responde 303 para el patrón Post/Redirect/Get.
func RedirectSeeOther(w http.ResponseWriter, r *http.Request, location string) {
	http.Redirect(w, r, location, http.StatusSeeOther)
}