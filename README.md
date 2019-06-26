# powerwall-homekit

HomeKit accessory for Tesla Powerwalls

# What does it do?

When launched, this will show up as a new bridge called "Tesla Bridge" which you can add to HomeKit.

Once added, it will also add two more accessories:

* `powerwall` - This is specified as an "other" type accessory, which HomeKit will say it doesn't support. It provides a "battery" HomeKit service which has charge level (percentage), charging state and low battery characteristics. Even though the official Home app won't let you, you can use 3rd party apps like HomeDash to create automations from the values of these characteristics.
* `grid` - This is a "sensor" accessory, with a contact sensor service. This provides an "open" or "closed" state to the sensor. Which in this case, is mapped to your grid power being connected or not. This will let you create automations around losing or regaining your grid power connection.