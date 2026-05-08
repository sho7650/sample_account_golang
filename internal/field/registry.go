package field

// Registry owns every Field, supports dispatch by short flag / long name,
// and exposes helpers used by the CLI parser and the --help printer.
type Registry struct {
	fields  []Field
	byShort map[byte]Field
	byLong  map[string]Field
}

func NewRegistry() *Registry {
	return &Registry{
		byShort: make(map[byte]Field),
		byLong:  make(map[string]Field),
	}
}

// Add registers a field. Long-name aliases are added separately via AddAlias.
func (r *Registry) Add(f Field) {
	r.fields = append(r.fields, f)
	r.byShort[f.ShortFlag()] = f
	r.byLong[f.LongName()] = f
}

// AddAlias makes alias resolve to the same field as the canonical long name.
func (r *Registry) AddAlias(alias, canonical string) {
	if f, ok := r.byLong[canonical]; ok {
		r.byLong[alias] = f
	}
}

func (r *Registry) FindShort(b byte) Field   { return r.byShort[b] }
func (r *Registry) FindLong(s string) Field  { return r.byLong[s] }
func (r *Registry) All() []Field             { return r.fields }

// ShortOptString returns the concatenation of every registered short flag,
// in registration order. Used by --help and the CLI parser.
func (r *Registry) ShortOptString() string {
	out := make([]byte, 0, len(r.fields))
	for _, f := range r.fields {
		out = append(out, f.ShortFlag())
	}
	return string(out)
}

// DefaultRegistry returns a registry populated with every field this binary
// supports, in the order they should appear in --help.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.Add(&IDField{})
	r.Add(&LastNameField{})
	r.Add(&FirstNameField{})
	r.Add(&MailField{})
	r.Add(&TelephoneField{})
	r.Add(&PrefectureField{})
	r.Add(&WardField{})
	r.Add(&CityField{})
	r.Add(&GenderField{})
	r.Add(&BloodField{})
	r.Add(&AgeField{})
	r.Add(&AgeGroupField{})
	r.Add(&BirthYearField{})
	r.Add(&RewardField{})
	r.Add(&DateField{})
	r.Add(&RandomIntField{})
	r.Add(&QuotientField{})
	r.AddAlias("telehpne", "telephone") // legacy typo preserved
	return r
}
