package field

import "strconv"

// All field types are tiny zero-sized structs. Emit appends bytes directly
// to the buffer using strconv.Append* — no intermediate string allocation.

// -----------------------------------------------------------------------------
// Identity
// -----------------------------------------------------------------------------

type IDField struct{}

func (IDField) ShortFlag() byte      { return 'i' }
func (IDField) LongName() string     { return "id" }
func (IDField) Description() string  { return "sequential row id (1-based)" }
func (IDField) Emit(buf []byte, ctx RowContext, _ Deps) []byte {
	return strconv.AppendInt(buf, int64(ctx.Row+1), 10)
}

// -----------------------------------------------------------------------------
// Person
// -----------------------------------------------------------------------------

type LastNameField struct{}

func (LastNameField) ShortFlag() byte     { return 'l' }
func (LastNameField) LongName() string    { return "lastname" }
func (LastNameField) Description() string { return "last name (kanji,kana — two CSV fields)" }
func (LastNameField) Emit(buf []byte, ctx RowContext, deps Deps) []byte {
	return append(buf, deps.Person.LastName(ctx.Last)...)
}

type FirstNameField struct{}

func (FirstNameField) ShortFlag() byte     { return 'f' }
func (FirstNameField) LongName() string    { return "firstname" }
func (FirstNameField) Description() string { return "first name (kanji,kana — two CSV fields)" }
func (FirstNameField) Emit(buf []byte, ctx RowContext, deps Deps) []byte {
	return append(buf, deps.Person.FirstName(ctx.First)...)
}

type MailField struct{}

func (MailField) ShortFlag() byte     { return 'm' }
func (MailField) LongName() string    { return "mail" }
func (MailField) Description() string { return "email address (firstname_lastname@example.com)" }
func (MailField) Emit(buf []byte, ctx RowContext, deps Deps) []byte {
	return deps.Person.AppendMailAddress(buf, ctx.First, ctx.Last)
}

type GenderField struct{}

func (GenderField) ShortFlag() byte     { return 'g' }
func (GenderField) LongName() string    { return "gender" }
func (GenderField) Description() string { return "gender (男 / 女)" }
func (GenderField) Emit(buf []byte, ctx RowContext, deps Deps) []byte {
	return append(buf, deps.Person.Gender(ctx.First)...)
}

type BloodField struct{}

func (BloodField) ShortFlag() byte     { return 'b' }
func (BloodField) LongName() string    { return "blood" }
func (BloodField) Description() string { return "ABO blood type" }
func (BloodField) Emit(buf []byte, _ RowContext, deps Deps) []byte {
	return append(buf, deps.Person.Blood(deps.Rng.Next())...)
}

type TelephoneField struct{}

func (TelephoneField) ShortFlag() byte     { return 't' }
func (TelephoneField) LongName() string    { return "telephone" }
func (TelephoneField) Description() string { return "phone number (090-XXXX-XXXX)" }
func (TelephoneField) Emit(buf []byte, _ RowContext, deps Deps) []byte {
	a := deps.Rng.Next() % 1000
	b := deps.Rng.Next() % 1000
	buf = append(buf, '0', '9', '0', '-')
	buf = appendPaddedInt(buf, a, 4)
	buf = append(buf, '-')
	buf = appendPaddedInt(buf, b, 4)
	return buf
}

// appendPaddedInt writes n into buf as a zero-padded width-w decimal.
// Negative or oversized values are clamped to non-negative and truncated.
func appendPaddedInt(buf []byte, n, w int) []byte {
	if n < 0 {
		n = -n
	}
	var tmp [16]byte
	end := len(tmp)
	if n == 0 {
		tmp[end-1] = '0'
		end--
	} else {
		for n > 0 {
			end--
			tmp[end] = byte('0' + n%10)
			n /= 10
		}
	}
	digits := len(tmp) - end
	for i := digits; i < w; i++ {
		buf = append(buf, '0')
	}
	return append(buf, tmp[end:]...)
}

// -----------------------------------------------------------------------------
// Address
// -----------------------------------------------------------------------------

type PrefectureField struct{}

func (PrefectureField) ShortFlag() byte     { return 'p' }
func (PrefectureField) LongName() string    { return "prefecture" }
func (PrefectureField) Description() string { return "prefecture name (population-weighted)" }
func (PrefectureField) Emit(buf []byte, ctx RowContext, deps Deps) []byte {
	return append(buf, deps.Address.PrefectureName(ctx.Pref)...)
}

type WardField struct{}

func (WardField) ShortFlag() byte     { return 'w' }
func (WardField) LongName() string    { return "ward" }
func (WardField) Description() string { return "ward / municipality within the prefecture" }
func (WardField) Emit(buf []byte, ctx RowContext, deps Deps) []byte {
	return append(buf, deps.Address.Ward(ctx.Pref, ctx.Ward)...)
}

type CityField struct{}

func (CityField) ShortFlag() byte     { return 'c' }
func (CityField) LongName() string    { return "city" }
func (CityField) Description() string { return "city / district within the ward" }
func (CityField) Emit(buf []byte, ctx RowContext, deps Deps) []byte {
	return append(buf, deps.Address.City(ctx.Pref, ctx.City)...)
}

// -----------------------------------------------------------------------------
// Age / date / numeric
// -----------------------------------------------------------------------------

type AgeField struct{}

func (AgeField) ShortFlag() byte     { return 'a' }
func (AgeField) LongName() string    { return "age" }
func (AgeField) Description() string { return "age in years (population-weighted)" }
func (AgeField) Emit(buf []byte, ctx RowContext, deps Deps) []byte {
	return strconv.AppendInt(buf, int64(deps.Age.Age(ctx.Age)), 10)
}

type AgeGroupField struct{}

func (AgeGroupField) ShortFlag() byte     { return 'o' }
func (AgeGroupField) LongName() string    { return "agegroup" }
func (AgeGroupField) Description() string { return "age group rounded down to the decade" }
func (AgeGroupField) Emit(buf []byte, ctx RowContext, deps Deps) []byte {
	return strconv.AppendInt(buf, int64(deps.Age.AgeGroup(ctx.Age)), 10)
}

type BirthYearField struct{}

func (BirthYearField) ShortFlag() byte     { return 'y' }
func (BirthYearField) LongName() string    { return "birthyear" }
func (BirthYearField) Description() string { return "birth year derived from age" }
func (BirthYearField) Emit(buf []byte, ctx RowContext, deps Deps) []byte {
	return strconv.AppendInt(buf, int64(deps.Age.BirthYear(ctx.Age)), 10)
}

type RewardField struct{}

func (RewardField) ShortFlag() byte     { return 'r' }
func (RewardField) LongName() string    { return "reward" }
func (RewardField) Description() string { return "annual income-like figure derived from age group" }
func (RewardField) Emit(buf []byte, ctx RowContext, deps Deps) []byte {
	return strconv.AppendInt(buf, int64(deps.Age.Reward(ctx.Age, deps.Rng)), 10)
}

type DateField struct{}

func (DateField) ShortFlag() byte     { return 'd' }
func (DateField) LongName() string    { return "date" }
func (DateField) Description() string { return "random valid date (YYYY/M/D)" }
func (DateField) Emit(buf []byte, _ RowContext, deps Deps) []byte {
	y, m, d := deps.Rng.RollDate()
	buf = strconv.AppendInt(buf, int64(y), 10)
	buf = append(buf, '/')
	buf = strconv.AppendInt(buf, int64(m), 10)
	buf = append(buf, '/')
	return strconv.AppendInt(buf, int64(d), 10)
}

type RandomIntField struct{}

func (RandomIntField) ShortFlag() byte     { return 'n' }
func (RandomIntField) LongName() string    { return "random" }
func (RandomIntField) Description() string { return "random signed integer in ±10,000,000" }
func (RandomIntField) Emit(buf []byte, _ RowContext, deps Deps) []byte {
	v := (deps.Rng.Next()%20001 - 10000) * 1000
	return strconv.AppendInt(buf, int64(v), 10)
}

type QuotientField struct{}

func (QuotientField) ShortFlag() byte     { return 'q' }
func (QuotientField) LongName() string    { return "quotient" }
func (QuotientField) Description() string { return "random fraction in [0.00, 0.99]" }
func (QuotientField) Emit(buf []byte, _ RowContext, deps Deps) []byte {
	v := deps.Rng.Next() % 100
	buf = append(buf, '0', '.')
	if v < 10 {
		buf = append(buf, '0')
	}
	return strconv.AppendInt(buf, int64(v), 10)
}
