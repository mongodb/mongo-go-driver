package insertopt

import "github.com/mongodb/mongo-go-driver/options-design/option"

type One interface{ one() }
type Many interface{ many() }

type OneBundle struct{}

func BundleOne(...One) *OneBundle { return nil }

func (ob *OneBundle) one()                                       {}
func (ob *OneBundle) BypassDocumentValidation(b bool) *OneBundle { return nil }
func (ob *OneBundle) Unbundle() []option.InsertOneOptioner       { return nil }

type ManyBundle struct{}

func BundleMany(...Many) *ManyBundle { return nil }

func (mb *ManyBundle) many()                                       {}
func (mb *ManyBundle) BypassDocumentValidation(b bool) *ManyBundle { return nil }
func (mb *ManyBundle) Ordered(b bool) *ManyBundle                  { return nil }
func (mb *ManyBundle) Unbundle() []option.InsertManyOptioner       { return nil }

func BypassDocumentValidation(b bool) OptBypassDocumentValidation { return false }
func Ordered(b bool) OptOrdered                                   { return false }

type OptBypassDocumentValidation option.OptBypassDocumentValidation
type OptOrdered option.OptOrdered

func (OptBypassDocumentValidation) one()  {}
func (OptBypassDocumentValidation) many() {}

func (OptOrdered) many() {}
