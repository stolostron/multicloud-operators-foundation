// licensed Materials - Property of IBM
// 5737-E67
// (C) Copyright IBM Corporation 2016, 2019 All Rights Reserved
// US Government Users Restricted Rights - Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.

package options

func (options *ServerRunOptions) Validate() []error {
	var errors []error
	if errs := options.GenericServerRunOptions.Validate(); len(errs) > 0 {
		errors = append(errors, errs...)
	}
	if errs := options.Etcd.Validate(); len(errs) > 0 {
		errors = append(errors, errs...)
	}
	if errs := options.SecureServing.Validate(); len(errs) > 0 {
		errors = append(errors, errs...)
	}
	if errs := options.Audit.Validate(); len(errs) > 0 {
		errors = append(errors, errs...)
	}
	if errs := options.Authentication.Validate(); len(errs) > 0 {
		errors = append(errors, errs...)
	}
	if errs := options.Authorization.Validate(); len(errs) > 0 {
		errors = append(errors, errs...)
	}
	// TODO: add more checks
	return errors
}
