// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// ActivityError Error returned from activity worker
// swagger:model activityError
type ActivityError struct {

	// Detailed explanation of error
	Details string `json:"details,omitempty"`

	// Summary explanation of error
	// Required: true
	Reason *string `json:"reason"`
}

// Validate validates this activity error
func (m *ActivityError) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateReason(formats); err != nil {
		// prop
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ActivityError) validateReason(formats strfmt.Registry) error {

	if err := validate.Required("reason", "body", m.Reason); err != nil {
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *ActivityError) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ActivityError) UnmarshalBinary(b []byte) error {
	var res ActivityError
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
