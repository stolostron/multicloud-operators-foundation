//go:build !ignore_autogenerated
// +build !ignore_autogenerated

// Code generated by deepcopy-gen. DO NOT EDIT.

package azure

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DiskEncryptionSet) DeepCopyInto(out *DiskEncryptionSet) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DiskEncryptionSet.
func (in *DiskEncryptionSet) DeepCopy() *DiskEncryptionSet {
	if in == nil {
		return nil
	}
	out := new(DiskEncryptionSet)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *MachinePool) DeepCopyInto(out *MachinePool) {
	*out = *in
	if in.Zones != nil {
		in, out := &in.Zones, &out.Zones
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.OSDisk.DeepCopyInto(&out.OSDisk)
	if in.OSImage != nil {
		in, out := &in.OSImage, &out.OSImage
		*out = new(OSImage)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new MachinePool.
func (in *MachinePool) DeepCopy() *MachinePool {
	if in == nil {
		return nil
	}
	out := new(MachinePool)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Metadata) DeepCopyInto(out *Metadata) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Metadata.
func (in *Metadata) DeepCopy() *Metadata {
	if in == nil {
		return nil
	}
	out := new(Metadata)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OSDisk) DeepCopyInto(out *OSDisk) {
	*out = *in
	if in.DiskEncryptionSet != nil {
		in, out := &in.DiskEncryptionSet, &out.DiskEncryptionSet
		*out = new(DiskEncryptionSet)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OSDisk.
func (in *OSDisk) DeepCopy() *OSDisk {
	if in == nil {
		return nil
	}
	out := new(OSDisk)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *OSImage) DeepCopyInto(out *OSImage) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OSImage.
func (in *OSImage) DeepCopy() *OSImage {
	if in == nil {
		return nil
	}
	out := new(OSImage)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Platform) DeepCopyInto(out *Platform) {
	*out = *in
	out.CredentialsSecretRef = in.CredentialsSecretRef
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Platform.
func (in *Platform) DeepCopy() *Platform {
	if in == nil {
		return nil
	}
	out := new(Platform)
	in.DeepCopyInto(out)
	return out
}
