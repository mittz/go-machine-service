package dynamic

import (
	"strings"

	"github.com/rancher/go-rancher/v3"
)

func UploadMachineSchemas(apiClient *client.RancherClient, drivers ...string) error {
	schemaLock.Lock()
	defer schemaLock.Unlock()

	existing, err := apiClient.MachineDriver.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"removed_null": "true",
		},
	})
	if err != nil {
		return err
	}

	for _, driver := range existing.Data {
		if driver.Name != "" && (driver.State == "active" || driver.State == "activating") {
			drivers = append(drivers, driver.Name)
		}
	}

	logger.Infof("Updating machine jsons for  %v", drivers)
	if err := uploadMachineServiceJSON(drivers, true); err != nil {
		return err
	}
	if err := uploadMachineProjectJSON(drivers, false); err != nil {
		return err
	}
	if err := uploadMachineUserJSON(drivers, false); err != nil {
		return err
	}
	return uploadMachineReadOnlyJSON(false)
}

func field(resourceFields map[string]client.Field, field, fieldType, auth string) {
	resourceFields[field] = client.Field{
		Nullable: true,
		Type:     fieldType,
		Create:   strings.Contains(auth, "c"),
		Update:   strings.Contains(auth, "u"),
	}
}

func baseSchema(drivers []string, defaultAuth string) client.Schema {
	schema := client.Schema{
		CollectionMethods: []string{"GET"},
		ResourceMethods:   []string{"GET"},
		ResourceFields: map[string]client.Field{
			"name": client.Field{
				Type:     "string",
				Create:   true,
				Update:   true,
				Required: false,
				Nullable: true,
			},
		},
	}

	if strings.Contains(defaultAuth, "c") {
		schema.CollectionMethods = append(schema.CollectionMethods, "POST")
		schema.ResourceMethods = append(schema.ResourceMethods, "DELETE")
	}

	if strings.Contains(defaultAuth, "u") {
		schema.ResourceMethods = append(schema.ResourceMethods, "PUT")
	}

	for _, driver := range drivers {
		field(schema.ResourceFields, driver+"Config", driver+"Config", defaultAuth)
	}

	return schema
}

func uploadMachineServiceJSON(drivers []string, remove bool) error {
	schema := baseSchema(drivers, "cu")
	field(schema.ResourceFields, "extractedConfig", "string", "u")
	field(schema.ResourceFields, "labels", "map[string]", "cu")

	return uploadMachineSchema(schema, []string{"service"}, remove)
}

func uploadMachineProjectJSON(drivers []string, remove bool) error {
	schema := baseSchema(drivers, "cu")
	return uploadMachineSchema(schema, []string{"project", "member", "owner"}, remove)
}

func uploadMachineUserJSON(drivers []string, remove bool) error {
	schema := baseSchema(drivers, "cu")
	return uploadMachineSchema(schema, []string{"admin", "user", "readAdmin"}, remove)
}

func uploadMachineReadOnlyJSON(remove bool) error {
	schema := baseSchema([]string{}, "")
	return uploadMachineSchema(schema, []string{"readonly"}, remove)
}

func uploadMachineSchema(schema client.Schema, roles []string, remove bool) error {
	json, err := toJSON(&schema)
	if err != nil {
		return err
	}
	err = uploadDynamicSchema("host", json, "host", roles, remove)
	return err
}
