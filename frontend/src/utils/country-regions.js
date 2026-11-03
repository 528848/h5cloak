import regions from '../../public/json/country-regions-data.json';

const codeLabelMap = Object.fromEntries(
  regions.flatMap((region) =>
    (region.children || []).map((item) => [item.value, item.label])
  )
);

export function getCountryLabel(code) {
  if (!code) {
    return '';
  }
  return codeLabelMap[String(code).toUpperCase()] || String(code);
}

export function formatCountryCodes(codes) {
  if (!Array.isArray(codes) || codes.length === 0) {
    return '-';
  }
  return codes.map((code) => getCountryLabel(code)).join(', ');
}

export function buildRegionPaths(codes) {
  if (!Array.isArray(codes) || codes.length === 0) {
    return [];
  }
  return codes
    .map((code) => {
      const region = regions.find((item) =>
        (item.children || []).some(
          (child) => child.value === String(code).toUpperCase()
        )
      );
      if (!region) {
        return null;
      }
      return [region.value, String(code).toUpperCase()];
    })
    .filter(Boolean);
}

export { regions as countryRegions };
