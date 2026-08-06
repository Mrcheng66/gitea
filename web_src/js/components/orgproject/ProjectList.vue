<script setup lang="ts">
import type {OrgProjectListRow} from './types.ts';

defineProps<{
  rows: OrgProjectListRow[],
  baseLink: string,
  archivedLabel: string,
  noDescription: string,
}>();
</script>

<template>
  <div v-if="rows.length" class="org-project-table-wrap">
    <table class="ui single line table org-project-table">
      <thead>
        <tr>
          <th>Project</th>
          <th v-for="field in rows[0]!.Fields" :key="field.Key">{{ field.Label }}</th>
          <th class="collapsing">Status</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="row in rows" :key="row.Project.ID">
          <td>
            <a class="tw-font-semibold" :href="`${baseLink}/${row.Project.Slug}`">{{ row.Project.Name }}</a>
            <div class="text light-2 tw-mt-1 org-project-description">{{ row.Project.Description || noDescription }}</div>
          </td>
          <td v-for="field in row.Fields" :key="field.Key">{{ field.Value }}</td>
          <td><span v-if="row.Project.Lifecycle === 'archived'" class="ui basic label">{{ archivedLabel }}</span></td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
