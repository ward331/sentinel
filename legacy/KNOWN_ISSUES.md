# SENTINEL V3 Known Issues

## Low Severity

- **[LOW] Some Tier 1 providers require manual key injection.** OpenSanctions and Global Fishing Watch take API keys via constructor, not from the config `keys` section.

- **[LOW] Setup wizard runs non-interactively in containers.** The terminal wizard requires stdin, which may not be available in Docker. Use `--config` with a pre-created config file instead.

## Notes

All HIGH and MEDIUM severity issues from the initial build have been resolved. The system is production-ready.
