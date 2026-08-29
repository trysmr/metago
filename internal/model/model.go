package model

type ObjectDefinition struct {
	APIName          string
	Label            string
	PluralLabel      string
	DeploymentStatus string
	SharingModel     string
	EnableActivities bool
	EnableReports    bool
	EnableHistory    bool
	Fields           []FieldDefinition
}

type FieldDefinition struct {
	APIName      string
	Label        string
	Type         string
	Required     bool
	Unique       bool
	ExternalID   bool
	Encrypted    bool
	TrackHistory bool
	Length       *int
	Precision    *int
	Scale        *int
	ReferenceTo  []string
	Relationship string
	Formula      string
}
