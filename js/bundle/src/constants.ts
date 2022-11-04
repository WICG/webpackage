export const B1 = 'b1';
export const B2 = 'b2';
export const DEFAULT_VERSION = B2;
export const APPROVED_VERSIONS = [B1, B2] as const;
export type FormatVersion = typeof APPROVED_VERSIONS[number];

export function isApprovedVersion(param: any): param is FormatVersion {
  return APPROVED_VERSIONS.includes(param);
}
