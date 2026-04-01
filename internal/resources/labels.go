package resources

const (
	LabelManagedBy = "app.kubernetes.io/managed-by"
	LabelPartOf    = "app.kubernetes.io/part-of"
	LabelInstance  = "app.kubernetes.io/instance"
	LabelComponent = "app.kubernetes.io/component"
	LabelTenant    = "supabase.codanael.io/tenant"

	ManagedByValue = "supabase-operator"
	PartOfValue    = "supabase"
)

func PlatformLabels(instance, component string) map[string]string {
	return map[string]string{
		LabelManagedBy: ManagedByValue,
		LabelPartOf:    PartOfValue,
		LabelInstance:  instance,
		LabelComponent: component,
	}
}

func TenantLabels(instance, tenantID, component string) map[string]string {
	labels := PlatformLabels(instance, component)
	labels[LabelTenant] = tenantID
	return labels
}

func SelectorLabels(instance, component string) map[string]string {
	return map[string]string{
		LabelInstance:  instance,
		LabelComponent: component,
	}
}
