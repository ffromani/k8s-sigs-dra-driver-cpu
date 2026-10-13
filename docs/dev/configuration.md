# Adding configuration tunables

The configurable behavior of the DRA CPU driver can be changed using two approaches:

- command line flags
- configuration file

The preferred way to add or extend the configurable behavior is through the
configuration file. The reasons why we prefer this approach are:

- user friendliness: adding options quickly makes the command line long and awkward
- auditable, reviewable artifact: the configuration file can be versioned, compared
  and verified more easily
- extensibility: it's usually easier to process configuration files with programs
  than command line options
- configuration files are easier to version and evolve compatibly than command
  line flags.

Configuration values are applied incrementally:

- the driver has compiled in defaults
- the defaults are overridden by the configuration file, resulting in a merged configuration
- the explicitly set command line flags override the merged configuration, resulting in the
  final configuration which will be used.

**NOTE**: only flags that are actually passed on the command line take precedence; the unset
flags inherit the prior value from the previous incremental layer (or the default).

The current configuration-file first approach has a pretty large consensus, and there is no
obvious reason to change again in the foreseeable future, but is not written in stone;
we may revisit this stance in the future should the circumstances change.

Adding command line options is however **NOT** prohibited, but should be approached
with a healthy dose of skepticism.

If you are trying to add configurable behavior to the driver, a few questions can
help your decision making process:

- does the behavior need to be configured at all? Can we just change the defaults?
- is the option more like a feature flag or feature gate or something users may genuinely
  decide to flip depending on their setup?
  should you need a feature flag/gate, please get in touch to discuss the best approach.
- some tunables are needed because of the nature of the driver, which wants to run
  as containerized process as main delivery vehicle. If your option fits in this category,
  you may need no driver option at all, but a new helm chart value or a manifest change.
- is your option a tunable knob of the current operational mode? if so it belongs
  in the configuration file.
- is your option an operational-mode switch - a flag that changes what the driver does
  rather than tune the running driver? if so, a command line flag is likely warranted.

If we need to add operational modes, we express them as subcommands (e.g.
`dracpu introspect metrics` instead of `dracpu --show-metrics`), using a lightweight
stdlib-based dispatch internal package rather than a full CLI framework
(see [#234](https://github.com/kubernetes-sigs/dra-driver-cpu/issues/234),
[#131](https://github.com/kubernetes-sigs/dra-driver-cpu/pull/131)).
An internal lightweight package serves our current and foreseeable future needs well.
We don't have yet the need for a much heavier package like `cobra`.

As a concrete example: we briefly introduced the `--show-metrics` command line flag.
But this was quickly determined as operational-mode switch - it prints
metadata and exits instead of running the driver. Exposing it in the configuration file
led to a silent failure: the value was loaded and logged, but the print-and-exit handler
ran before the configuration file was read, so the setting was silently inert on the Helm path
(see [#234](https://github.com/kubernetes-sigs/dra-driver-cpu/issues/234)).

Please make sure to document your rationale (no need to write an essay: a few clear sentences
are perfectly fine) in your issue/PR so we can evaluate and converge towards the best shape.

## Handling multiple configuration version

The design uses public versioned configuration schemas and one internal runtime
configuration. [scheduler-plugins](https://github.com/kubernetes-sigs/scheduler-plugins/tree/master/apis/config)
is our **conceptual reference model** for this separation. We will use the simplest
implementation that fits this driver's needs, which will likely differ from that
reference: a small version dispatcher and explicit handwritten conversions suffice;
Kubernetes schemes and generated conversion machinery are not requirements.

Each supported schema converts directly to the internal configuration, without
passing through other public versions. Runtime code consumes only the internal
configuration and does not branch on the input version. Reverse conversion is only
needed if we introduce configuration export or migration tooling.

The loading sequence is:

1. Identify the version and strictly decode its public schema.
2. Apply that version's defaults and convert to the internal configuration.
3. Apply explicitly supplied CLI overrides, normalize, and validate the effective
   configuration.

Defaults for public fields belong to their schema version, not to core runtime
code. Older schemas must also supply deliberate compatibility values for internal
settings they cannot express. This preserves the meaning of both supplied and
omitted fields; accepting an old file is insufficient if its behavior changes
silently. CLI-only settings retain separate defaults. Without a config file, use
an empty instance of the current public schema through the same defaulting and
conversion path. The resolved internal configuration is complete, with no further
configuration defaulting in runtime code.

Decoding must reject unknown fields, duplicate keys, and incorrect key casing.
Define omission, explicit zero or empty values, and `null` deliberately; use pointer
fields where presence affects semantics. Versionless files always mean legacy
`v1alpha1`, never the latest schema. New formats require an explicit `apiVersion`.

We support the current schema and **at most two preceding schemas**, ordered by
introduction, independently of driver releases. Adding `v1alpha3` retains
`v1alpha1` and `v1alpha2`; adding `v1beta1` then retires `v1alpha1`. Releases without
a new schema do not change the support window. Retirement also ends support for
versionless files when `v1alpha1` leaves the window. Unsupported versions fail
startup with the supported versions and migration guidance. While a schema is
supported, the runtime must retain the behavior necessary to honor it.

Helm validation must derive from the public schemas and accept every supported
version. Rendering and deprecated argument folding must respect the selected
schema, without silently changing its version. New examples and generated files
must carry an explicit version. Document migration steps and chart/image
compatibility, including rollback: an older binary generally cannot read a newer
schema, so retain a mutually supported format until deliberately migrating.

Tests must cover historical defaults and behavior, equivalent conversions across
versions, field presence and strict decoding, explicit CLI precedence, retirement
errors, and Helm validation and rendering for every supported schema.
