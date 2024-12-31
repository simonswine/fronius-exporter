# Fronius-Exporter

Uses the fronious json v1 API to expose inverter metrics

## Examples

```
# HELP fronius_inverter_info Information about the inverter
# TYPE fronius_inverter_info gauge
fronius_inverter_info{device_id="1",device_name="hearts",device_type="232",serial="1234567"} 1
# HELP fronius_inverter_status Status of the inverter
# TYPE fronius_inverter_status gauge
fronius_inverter_status{device_id="1",status="bootloading"} 0
fronius_inverter_status{device_id="1",status="error"} 0
fronius_inverter_status{device_id="1",status="idle"} 0
fronius_inverter_status{device_id="1",status="invalid"} 0
fronius_inverter_status{device_id="1",status="ready"} 0
fronius_inverter_status{device_id="1",status="running"} 1
fronius_inverter_status{device_id="1",status="sleeping"} 0
fronius_inverter_status{device_id="1",status="standby"} 0
fronius_inverter_status{device_id="1",status="startup"} 0
fronius_inverter_status{device_id="1",status="unknown"} 0
# HELP inverter_dc_current Solar panel (DC) current
# TYPE inverter_dc_current gauge
inverter_dc_current{device_id="1"} 0.97
# HELP inverter_dc_voltage Solar panel (DC) voltage
# TYPE inverter_dc_voltage gauge
inverter_dc_voltage{device_id="1"} 636.9
# HELP inverter_grid_current Grid (AC) current
# TYPE inverter_grid_current gauge
inverter_grid_current{device_id="1",phase="L1"} 0.57
inverter_grid_current{device_id="1",phase="L2"} 0.71
inverter_grid_current{device_id="1",phase="L3"} 0.72
# HELP inverter_grid_frequency Grid (AC) frequency
# TYPE inverter_grid_frequency gauge
inverter_grid_frequency{device_id="1"} 49.97
# HELP inverter_grid_voltage Grid (AC) current
# TYPE inverter_grid_voltage gauge
inverter_grid_voltage{device_id="1",phase="L1"} 236.2
inverter_grid_voltage{device_id="1",phase="L2"} 234.9
inverter_grid_voltage{device_id="1",phase="L3"} 236.4
# HELP inverter_yield_total Information about the inverter
# TYPE inverter_yield_total counter
inverter_yield_total{device_id="1"} 234611.6
```
