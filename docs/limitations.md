---
title: "Limitations"
weight: 30
---

## Limitations of first use case

- Our focus is currently on collecting energy consumption metrics from SNMP 
  endpoints of PDUs and bare metals, while our vision is also on collecting 
  usage metrics of VMs from hypervisors and for pods in clusters.
- The current resource definitions of data center resources, physical 
  measurements and measurement devices have no simple means of indicating which 
  outlet of a PDU is involved (leads to duplicate scrape endpoint physical 
  measurement resources). We plan to support some kind of mapping from 
  connected PDU to bare metal server.
- The aforementioned mapping should be generic enough to allow reuse in 
  mappings from timeseries related to VMs, to allow translating internal unique 
  IDs from hypervisors to service IDs provided by the orchestrator.
- Our first use case focuses on providing time series streams of energy 
  consumption of bare metals based on measurements on PDU and bare metals. When 
  we are able to monitor both connected PDUs and the involved bare metal, there 
  may be differences in the energy measurements. These can be assigned to 
  measurement errors, exclusion of onboard BMC (IPMI/RAC "mini servers" hosted 
  in the same server unit) or other interfaces. A fair algorithm that 
  determines how to allocate the overhead to the data center operator and/or 
  server consumer is to be designed.
