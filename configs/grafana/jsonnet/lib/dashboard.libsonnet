local g = import '../g.libsonnet';

{
  new(title, uid, version, tags, panels, timeFrom='now-1h'):
    {
      id: null,
      links: [],
      templating: { list: [] },
      annotations: { list: [] },
      fiscalYearStartMonth: 0,
      graphTooltip: 0,
    }
    + g.dashboard.withTitle(title)
    + g.dashboard.withUid(uid)
    + g.dashboard.withEditable(true)
    + g.dashboard.withRefresh('10s')
    + g.dashboard.withSchemaVersion(39)
    + g.dashboard.withTags(tags)
    + g.dashboard.withPanels(panels)
    + g.dashboard.time.withFrom(timeFrom)
    + g.dashboard.time.withTo('now')
    + { version: version },
}
