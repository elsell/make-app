# Security policy

Please report suspected vulnerabilities privately through GitHub's security
advisory form for this repository. Do not open a public issue with exploit
details, credentials, personal data, or unredacted logs.

`make-app` is currently a `v0.x` project. Security fixes are provided for the
latest tagged minor release and current `main`; older generator releases and
generated template schemas receive no guaranteed backports. A generated
application owns its source and must apply generator guidance or patches itself.

Include the generator version, template schema from `.make-app.json`, host OS,
affected command or generated boundary, reproduction steps, and impact. We will
acknowledge a report as soon as practical, validate it privately, and coordinate
disclosure after a fix is available.
