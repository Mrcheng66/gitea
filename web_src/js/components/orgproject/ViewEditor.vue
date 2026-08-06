<script setup lang="ts">
import type {OrgProjectField, OrgProjectFilter, OrgProjectListView} from './types.ts';

const props = defineProps<{fields: OrgProjectField[]}>();
const listView = defineModel<OrgProjectListView>('listView', {required: true});
const filters = defineModel<OrgProjectFilter[]>('filters', {required: true});

function addSort() {
  const field = props.fields.find((item) => !item.archived);
  if (!field) return;
  listView.value.sort ??= [];
  listView.value.sort.push({field_key: field.key, direction: 'asc'});
}

function addFilter() {
  const field = props.fields.find((item) => !item.archived);
  if (!field) return;
  const index = filters.value.length + 1;
  filters.value.push({key: `filter_${index}`, label: `Filter ${index}`, field_key: field.key, operator: 'eq'});
}
</script>

<template>
  <section class="org-project-editor-section">
    <h3 class="ui header">List view</h3>
    <div class="field"><label>Columns</label><select v-model="listView.columns" multiple><option v-for="field in fields.filter((item) => !item.archived)" :key="field.key" :value="field.key">{{ field.label }}</option></select></div>
    <div class="flex-left-right tw-mt-4"><h4 class="ui header tw-m-0">Default sorting</h4><button type="button" class="ui small button" @click="addSort">Add sort</button></div>
    <div v-for="(sort, index) in listView.sort" :key="index" class="org-project-editor-grid tw-mt-3">
      <div class="field"><label>Field</label><select v-model="sort.field_key"><option v-for="field in fields.filter((item) => !item.archived)" :key="field.key" :value="field.key">{{ field.label }}</option></select></div>
      <div class="field"><label>Direction</label><select v-model="sort.direction"><option value="asc">Ascending</option><option value="desc">Descending</option></select></div>
      <button type="button" class="ui icon basic red button org-project-remove-button" aria-label="Remove sort" @click="listView.sort!.splice(index, 1)">×</button>
    </div>
    <div class="flex-left-right tw-mt-4"><h3 class="ui header tw-m-0">Filters</h3><button type="button" class="ui small button" @click="addFilter">Add filter</button></div>
    <div v-for="(filter, index) in filters" :key="index" class="org-project-editor-grid tw-mt-3">
      <div class="field"><label>Key</label><input v-model.trim="filter.key"></div>
      <div class="field"><label>Label</label><input v-model.trim="filter.label"></div>
      <div class="field"><label>Field</label><select v-model="filter.field_key"><option v-for="field in fields.filter((item) => !item.archived)" :key="field.key" :value="field.key">{{ field.label }}</option></select></div>
      <div class="field"><label>Operator</label><select v-model="filter.operator"><option value="eq">Equals</option><option value="ne">Not equal</option><option value="contains">Contains</option><option value="is_empty">Is empty</option><option value="is_not_empty">Is not empty</option><option value="gte">At least</option><option value="lte">At most</option><option value="member">Contains member</option></select></div>
      <button type="button" class="ui icon basic red button org-project-remove-button" aria-label="Remove filter" @click="filters.splice(index, 1)">×</button>
    </div>
  </section>
</template>
