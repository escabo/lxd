(instances-configure)=
# How to configure instances

You can configure instances by setting {ref}`instance-properties`, {ref}`instance-options`, or by adding and configuring {ref}`devices`.

See the following sections for instructions.

```{note}
To store and reuse different instance configurations, use {ref}`profiles <profiles>`.
```

(instances-configure-options)=
## Configure instance options

You can specify instance options when you {ref}`create an instance <instances-create>`.
Alternatively, you can update the instance options after the instance is created.

````{tabs}
```{group-tab} CLI
Use the [`lxc config set`](lxc_config_set.md) command to update instance options.
Specify the instance name and the key and value of the instance option:

    lxc config set <instance_name> <option_key>=<option_value> <option_key>=<option_value> ...
```

```{group-tab} API
Send a PATCH request to the instance to update instance options.
Specify the instance name and the key and value of the instance option:

    lxc query --request PATCH /1.0/instances/<instance_name> --data '{
      "config": {
        "<option_key>": "<option_value>",
        "<option_key>": "<option_value>"
      }
    }'

See [`PATCH /1.0/instances/{name}`](swagger:/instances/instance_patch) for more information.
```

```{group-tab} UI
To update instance options, go to the {guilabel}`Configuration` tab of the instance detail page and click {guilabel}`Edit instance`.

Find the configuration option that you want to update and change its value.
Click {guilabel}`Save changes` to save the updated configuration.

To configure instance options that are not displayed in the UI, follow the instructions in {ref}`instances-configure-edit`.
```
````

See {ref}`instance-options` for a list of available options and information about which options are available for which instance type.

For example, change the memory limit for your container:

`````{tabs}
```{group-tab} CLI
To set the memory limit to 8 GiB, enter the following command:

    lxc config set my-container limits.memory=8GiB
```

```{group-tab} API
To set the memory limit to 8 GiB, send the following request:

    lxc query --request PATCH /1.0/instances/my-container --data '{
      "config": {
        "limits.memory": "8GiB"
      }
    }'
```

````{group-tab} UI
To set the memory limit to 8 GiB, go to the {guilabel}`Configuration` tab of the instance detail page and select {guilabel}`Advanced > Resource limits`.
Then click {guilabel}`Edit instance`.

Select {guilabel}`Override` for the **Memory limit** and enter 8 GiB as the absolute value.

```{figure} /images/UI/limits_memory_example.png
   :width: 80%
   :alt: Setting the memory limit for an instance to 8 GiB
```
````
`````

```{note}
Some of the instance options are updated immediately while the instance is running.
Others are updated only when the instance is restarted.

See the "Live update" information in the {ref}`instance-options` reference for information about which options are applied immediately while the instance is running.
```

(instances-configure-properties)=
## Configure instance properties

````{tabs}
```{group-tab} CLI
To update instance properties after the instance is created, use the [`lxc config set`](lxc_config_set.md) command with the `--property` flag.
Specify the instance name and the key and value of the instance property:

    lxc config set <instance_name> <property_key>=<property_value> <property_key>=<property_value> ... --property

Using the same flag, you can also unset a property just like you would unset a configuration option:

    lxc config unset <instance_name> <property_key> --property

You can also retrieve a specific property value with:

    lxc config get <instance_name> <property_key> --property
```

```{group-tab} API
To update instance properties through the API, use the same mechanism as for configuring instance options.
The only difference is that properties are on the root level of the configuration, while options are under the `config` field.

Therefore, to set an instance property, send a PATCH request to the instance:

    lxc query --request PATCH /1.0/instances/<instance_name> --data '{
      "<property_key>": "<property_value>",
      "<property_key>": "property_value>"
      }
    }'

To unset an instance property, send a PUT request that contains the full instance configuration that you want except for the property that you want to unset.

See [`PATCH /1.0/instances/{name}`](swagger:/instances/instance_patch) and [`PUT /1.0/instances/{name}`](swagger:/instances/instance_put) for more information.
```

```{group-tab} UI
The LXD UI does not distinguish between instance options and instance properties.
Therefore, you can configure instance properties in the same way as you {ref}`configure instance options <instances-configure-options>`.
```
````

(instances-configure-devices)=
## Configure devices

Generally, devices can be added or removed for a container while it is running.
VMs support hotplugging for some device types, but not all.

See {ref}`devices` for a list of available device types and their options.

```{note}
Every device entry is identified by a name unique to the instance.

Devices from profiles are applied to the instance in the order in which the profiles are assigned to the instance.
Devices defined directly in the instance configuration are applied last.
At each stage, if a device with the same name already exists from an earlier stage, the whole device entry is overridden by the latest definition.

Device names are limited to a maximum of 64 characters.
```

`````{tabs}
````{group-tab} CLI
To add and configure an instance device for your instance, use the [`lxc config device add`](lxc_config_device_add.md) command.

Specify the instance name, a device name, the device type and maybe device options (depending on the {ref}`device type <devices>`):

    lxc config device add <instance_name> <device_name> <device_type> <device_option_key>=<device_option_value> <device_option_key>=<device_option_value> ...

For example, to add the storage at `/share/c1` on the host system to your instance at path `/opt`, enter the following command:

    lxc config device add my-container disk-storage-device disk source=/share/c1 path=/opt

To configure instance device options for a device that you have added earlier, use the [`lxc config device set`](lxc_config_device_set.md) command:

    lxc config device set <instance_name> <device_name> <device_option_key>=<device_option_value> <device_option_key>=<device_option_value> ...

Device options for a device inherited from a profile cannot be updated within the instance. Use the [`lxc config device override`](lxc_config_device_override.md) command to create a copy of the profile device with updated device options. The newly created instance device will override the inherited device.

Specify the instance name, device name and the device options that should be overridden:

    lxc config device override <instance_name> <device_name> <device_option_key>=<device_option_value> <device_option_key>=<device_option_value> ...

```{note}
You can also specify device options by using the `--device` flag when {ref}`creating an instance <instances-create>`.
This is useful if you want to override device options for a device that is provided through a {ref}`profile <profiles>`.
```

To remove a device, use the [`lxc config device remove`](lxc_config_device_remove.md) command.
See [`lxc config device --help`](lxc_config_device.md) for a full list of available commands.
````

````{group-tab} API
To add or configure an instance device for your instance, use the same mechanism of patching the instance configuration.
The device configuration is located under the `devices` field of the configuration.

```{caution}
Patching a device's configuration unsets any omitted options for that device, along with the instance's `description` property. See {ref}`instances-configure-devices-api-patch-effects` for details.
```

Specify the instance name, a device name, and any {ref}`instances-configure-devices-api-required` (depending on the {ref}`device type <devices>`):

    lxc query --request PATCH /1.0/instances/<instance_name> --data '{
      "devices": {
        "<device_name>": {
          "type": "<device_type>",
          "<device_option_key>": "<device_option_value>",
          "<device_option_key>": "device_option_value>"
        }
      }
    }'

For example, to add the storage at `/share/c1` on the host system to your instance at path `/opt`, enter the following command:

    lxc query --request PATCH /1.0/instances/my-container --data '{
      "devices": {
        "disk-storage-device": {
          "type": "disk",
          "source": "/share/c1",
          "path": "/opt"
        }
      }
    }'

See [`PATCH /1.0/instances/{name}`](swagger:/instances/instance_patch) for more information.

(instances-configure-devices-api-required)=
### Required device options

When using a PATCH request to update an instance's `devices` property, you must include any required options for each device in the request body. The device's `type` option is always required. To find any other required keys for a specific device type, view the {ref}`devices` reference guides. For example, for an OVN NIC device, the {config:option}`device-nic-ovn-device-conf:network` key is required.

(instances-configure-devices-api-patch-effects)=
### Effects of patching device options

For any device in your PATCH request, the request acts similar to a conventional PUT: it replaces all options for that device. This means that if you omit a non-required option, it is unset. Thus, include not only the options you want to add or update in your patch, but also any other existing options whose values you want to keep.

This behavior only affects the specific device or devices that you are patching; if there are other devices, you don't need to include them. It also does not affect any other instance properties, with one exception: if the instance includes a `description` property, that property must be passed along with `devices`; otherwise, it is unset.

For example, consider an instance that contains this `devices` property:

```bash
"devices": {
  "my-bridge-nic": {
    "name": "my-bridge-nic-name",
    "network": "my-bridge-network",
    "type": "nic"
  },
  "my-ovn-nic": {
    "name": "my-ovn-nic-name",
    "network": "my-ovn-network",
    "type": "nic"
  }
}
```

Let's say the following PATCH request is sent for this instance:

```bash
lxc query --request PATCH /1.0/instances/my-instance --data '{
  "devices": {
    "my-bridge-nic": {
      "type": "nic",
      "network": "test-bridge",
      "ipv4.address": "192.0.2.10"
    }
  }
}'
```

This PATCH request updates only the `my-bridge-nic` device, without affecting the `my-ovn-nic` device. The device options defined in the request body replace the existing options. After the request, this is the `devices` property's configuration:

```bash
"devices": {
  "my-bridge-nic": {
    "network": "my-bridge-network",
    "type": "nic",
    "ipv4.address": "192.0.2.10"
  },
  "my-ovn-nic": {
    "name": "my-ovn-nic-name",
    "network": "my-ovn-network",
    "type": "nic"
  }
}
```

Notice that in the updated `my-bridge-nic` device, the `name` option is unset and no longer appears, due to not being sent in the PATCH request.

````

````{group-tab} UI
The UI does not support all device types yet, but you can configure disk and network devices for your instances.

To attach a device to your instance, or modify an existing device, update your instance configuration (in the same way as you {ref}`configure instance options <instances-configure-options>`).
Select {guilabel}`Advanced` > {guilabel}`Disk devices` > {guilabel}`Attach disk device` or {guilabel}`Advanced` > {guilabel}`Network devices` > {guilabel}`Attach network` to create a device and attach it to your instance.

```{note}
Some of the devices that are displayed in the instance configuration are inherited from a {ref}`profile <profiles>` or defined through a {ref}`project <projects>`.
Depending on the type of device, it might not be possible to edit these devices for an instance.
```

To add and configure devices that are not currently supported in the UI, follow the instructions in {ref}`instances-configure-edit`.
````

`````

## Display instance configuration

````{tabs}
```{group-tab} CLI
To display the current configuration of your instance, including writable instance properties, instance options, devices and device options, enter the following command:

    lxc config show <instance_name> --expanded
```

```{group-tab} API
To retrieve the current configuration of your instance, including writable instance properties, instance options, devices and device options, send a GET request to the instance:

    lxc query --request GET /1.0/instances/<instance_name>

See [`GET /1.0/instances/{name}`](swagger:/instances/instance_get) for more information.
```

```{group-tab} UI
To view the current configuration of your instance, go to {guilabel}`Instances`, select your instance, and then switch to the {guilabel}`Configuration` tab.

To see the full configuration including instance properties, instance options, devices and device options (also the ones that aren't yet supported by the UI), select {guilabel}`YAML configuration`.
This view shows the full YAML of the instance configuration.
```
````

(instances-configure-edit)=
## Edit the full instance configuration

`````{tabs}
````{group-tab} CLI
To edit the full instance configuration, including writable instance properties, instance options, devices and device options, enter the following command:

    lxc config edit <instance_name>

```{note}
For convenience, the [`lxc config edit`](lxc_config_edit.md) command displays the full configuration including read-only instance properties.
However, you cannot edit those properties.
Any changes are ignored.
```
````

````{group-tab} API
To update the full instance configuration, including writable instance properties, instance options, devices and device options, send a PUT request to the instance:

    lxc query --request PUT /1.0/instances/<instance_name> --data '<instance_configuration>'

See [`PUT /1.0/instances/{name}`](swagger:/instances/instance_put) for more information.

```{note}
If you include changes to any read-only instance properties in the configuration you provide, they are ignored.
```
````

````{group-tab} UI
Instead of using the UI forms to configure your instance, you can choose to edit the YAML configuration of the instance.
You must use this method if you need to update any configurations that are not available in the UI.

```{important}
When doing updates, do not navigate away from the YAML configuration without saving your changes.
If you do, your updates are lost.
```

To edit the YAML configuration of your instance, go to the instance detail page, switch to the {guilabel}`Configuration` tab and select {guilabel}`YAML configuration`.
Then click {guilabel}`Edit instance`.

Edit the YAML configuration as required.
Then click {guilabel}`Save changes` to save the updated configuration.

```{note}
For convenience, the YAML contains the full configuration including read-only instance properties.
However, you cannot edit those properties.
Any changes are ignored.
```
````
`````
