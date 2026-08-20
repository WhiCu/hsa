package domain

import (
	"testing"
)

type ValidStruct struct {
	Name string `validate:"required"`
	Age  int    `validate:"gte=0,lte=130"`
}

func TestValidateStruct(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name: "Valid struct",
			input: ValidStruct{
				Name: "John Doe",
				Age:  30,
			},
			wantErr: false,
		},
		{
			name: "Invalid struct - missing required",
			input: ValidStruct{
				Age: 30,
			},
			wantErr: true,
		},
		{
			name: "Invalid struct - out of range",
			input: ValidStruct{
				Name: "Jane Doe",
				Age:  150,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStruct() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
