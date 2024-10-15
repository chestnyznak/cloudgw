# VPP test notes

- Test VPP repository functions (including `ClearVPPConfig` and `AddVPPConfig`) with VPP container.
- VPP containers don't support Apple Silicon
- `vpp_startup.conf` should be as original (default) one except disabled `dpdk_plugin.so` to avoid CI workers/runners crash:

    ```bash
    plugins {
        plugin dpdk_plugin.so { disable }
    }
    ```
