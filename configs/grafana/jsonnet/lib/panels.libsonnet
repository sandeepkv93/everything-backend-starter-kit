{
  timeseries(
    id,
    title,
    x,
    y,
    w,
    h,
    datasourceType,
    datasourceUid,
    expr,
    legendFormat='A',
    unit=null,
    min=null,
    max=null,
    queryType=null,
  ):
    {
      id: id,
      title: title,
      type: 'timeseries',
      gridPos: { x: x, y: y, w: w, h: h },
      datasource: { type: datasourceType, uid: datasourceUid },
      options: { legend: { displayMode: 'list', placement: 'bottom' } },
      fieldConfig: {
        defaults:
          {}
          + (if unit == null then {} else { unit: unit })
          + (if min == null then {} else { min: min })
          + (if max == null then {} else { max: max }),
        overrides: [],
      },
      targets: [
        {
          refId: 'A',
          expr: expr,
        }
        + (if legendFormat == null then {} else { legendFormat: legendFormat })
        + (if queryType == null then {} else { queryType: queryType }),
      ],
    },

  stat(
    id,
    title,
    x,
    y,
    w,
    h,
    datasourceType,
    datasourceUid,
    expr,
    unit=null,
    legendFormat=null,
    queryType=null,
  ):
    {
      id: id,
      title: title,
      type: 'stat',
      gridPos: { x: x, y: y, w: w, h: h },
      datasource: { type: datasourceType, uid: datasourceUid },
      options: {
        colorMode: 'value',
        graphMode: 'area',
        justifyMode: 'auto',
        orientation: 'auto',
        reduceOptions: { calcs: ['lastNotNull'], fields: '', values: false },
      },
      fieldConfig: {
        defaults: {}
          + (if unit == null then {} else { unit: unit }),
        overrides: [],
      },
      targets: [
        {
          refId: 'A',
          expr: expr,
        }
        + (if legendFormat == null then {} else { legendFormat: legendFormat })
        + (if queryType == null then {} else { queryType: queryType }),
      ],
    },

  logs(id, title, x, y, w, h, expr, queryType=null):
    {
      id: id,
      title: title,
      type: 'logs',
      gridPos: { x: x, y: y, w: w, h: h },
      datasource: { type: 'loki', uid: 'loki' },
      options: {
        dedupStrategy: 'none',
        enableLogDetails: true,
        prettifyLogMessage: true,
        showCommonLabels: false,
        showLabels: false,
        showTime: true,
        sortOrder: 'Descending',
      },
      targets: [
        {
          refId: 'A',
          expr: expr,
        }
        + (if queryType == null then {} else { queryType: queryType }),
      ],
    },
}
