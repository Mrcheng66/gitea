<script setup lang="ts">
import type {OrgProjectListRow} from './types.ts';

defineProps<{
  rows: OrgProjectListRow[],
  baseLink: string,
  archivedLabel: string,
  noDescription: string,
  projectLabel: string,
  updatedLabel: string,
}>();

function initials(name: string) {
  return Array.from(name.trim())[0]?.toUpperCase() || '?';
}

function memberName(member: {name: string, full_name: string}) {
  return member.full_name || member.name;
}

function statusClass(key: string, raw: string) {
  if (key === 'risk') return `org-project-status is-risk-${raw || 'unknown'}`;
  if (key === 'stage') return `org-project-status is-stage-${raw || 'unknown'}`;
  return '';
}

function progress(field: OrgProjectListRow['Fields'][number]) {
  return Math.min(100, Math.max(0, field.Number || 0));
}

function updatedDate(timestamp: number) {
  return new Date(timestamp * 1000).toLocaleDateString(undefined, {month: 'short', day: 'numeric'});
}
</script>

<template>
  <div v-if="rows.length" class="org-project-table-wrap">
    <table class="ui table org-project-table">
      <thead>
        <tr>
          <th>{{ projectLabel }}</th>
          <th v-for="field in rows[0]!.Fields" :key="field.Key">{{ field.Label }}</th>
          <th class="collapsing">{{ updatedLabel }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="row in rows" :key="row.Project.ID">
          <td class="org-project-name-cell">
            <a class="tw-font-semibold" :href="`${baseLink}/${row.Project.Slug}`">{{ row.Project.Name }}</a>
            <span v-if="row.Project.Lifecycle === 'archived'" class="ui tiny basic label tw-ml-2">{{ archivedLabel }}</span>
            <div class="text light-2 tw-mt-1 org-project-description">{{ row.Project.Description || noDescription }}</div>
          </td>
          <td v-for="field in row.Fields" :key="field.Key" :class="`org-project-field-${field.Key}`">
            <div v-if="field.Type === 'percent'" class="org-project-progress" :aria-label="field.Value">
              <span><i :style="{width: `${progress(field)}%`}"/></span><b>{{ field.Value }}</b>
            </div>
            <div v-else-if="field.Type === 'member' && field.Members?.length" class="org-project-owner">
              <i class="org-project-avatar">{{ initials(memberName(field.Members[0]!)) }}</i>
              <span>{{ memberName(field.Members[0]!) }}</span>
            </div>
            <div v-else-if="field.Type === 'member_array' && field.Members?.length" class="org-project-followers" :title="field.Members.map(memberName).join('、')">
              <i v-for="member in field.Members.slice(0, 3)" :key="member.id" class="org-project-avatar">{{ initials(memberName(member)) }}</i>
              <i v-if="field.Members.length > 3" class="org-project-avatar is-count">+{{ field.Members.length - 3 }}</i>
            </div>
            <span v-else-if="field.Key === 'stage' || field.Key === 'risk'" :class="statusClass(field.Key, field.Raw)">{{ field.Value }}</span>
            <span v-else>{{ field.Value }}</span>
          </td>
          <td class="org-project-updated">{{ updatedDate(row.Project.UpdatedUnix) }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
