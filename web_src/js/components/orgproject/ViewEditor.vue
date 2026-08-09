<script setup lang="ts">
import {computed} from 'vue';
import type {OrgProjectField, OrgProjectFilter, OrgProjectListView} from './types.ts';

const props = defineProps<{fields: OrgProjectField[], labels: Record<string, string>}>();
const listView = defineModel<OrgProjectListView>('listView', {required: true});
const filters = defineModel<OrgProjectFilter[]>('filters', {required: true});
const activeFields = computed(() => props.fields.filter((item) => !item.archived));

function label(key: string) {
  return props.labels[key] || key;
}

function format(key: string, value: number) {
  return label(key).replace('{n}', String(value));
}

function addSort() {
  const field = activeFields.value[0];
  if (!field) return;
  listView.value.sort ??= [];
  listView.value.sort.push({field_key: field.key, direction: 'asc'});
}

function addFilter() {
  const field = activeFields.value[0];
  if (!field) return;
  const index = filters.value.length + 1;
  filters.value.push({key: `filter_${index}`, label: format('newFilter', index), field_key: field.key, operator: 'eq'});
}
</script>

<template>
  <section id="org-project-settings-list-view" class="ui segment org-project-settings-panel" aria-labelledby="org-project-settings-list-view-title">
    <header class="org-project-settings-panel-heading">
      <div>
        <h3 id="org-project-settings-list-view-title">{{ label('listView') }}</h3>
        <p>{{ label('listViewDescription') }}</p>
      </div>
    </header>
    <div class="org-project-settings-panel-body">
      <div class="field">
        <label>{{ label('columns') }}</label>
        <p class="help">{{ label('columnsDescription') }}</p>
        <div class="org-project-column-options">
          <label v-for="field in activeFields" :key="field.key" class="org-project-column-option" :class="{'is-selected': listView.columns.includes(field.key)}">
            <input v-model="listView.columns" type="checkbox" :value="field.key">
            <span>{{ field.label }}</span>
          </label>
        </div>
      </div>
      <div class="org-project-field-subheading">
        <strong>{{ label('defaultSorting') }}</strong>
        <button type="button" class="ui small button" :disabled="!activeFields.length" @click="addSort">{{ label('addSort') }}</button>
      </div>
      <div v-if="listView.sort?.length" class="org-project-rule-list">
        <div v-for="(sort, index) in listView.sort" :key="index" class="org-project-rule org-project-sort-rule">
          <span class="org-project-rule-index">{{ index + 1 }}</span>
          <div class="field"><label>{{ label('field') }}</label><select v-model="sort.field_key"><option v-for="field in activeFields" :key="field.key" :value="field.key">{{ field.label }}</option></select></div>
          <div class="field"><label>{{ label('direction') }}</label><select v-model="sort.direction"><option value="asc">{{ label('ascending') }}</option><option value="desc">{{ label('descending') }}</option></select></div>
          <button type="button" class="ui tiny basic red button org-project-rule-remove" :aria-label="label('removeSort')" @click="listView.sort!.splice(index, 1)">{{ label('remove') }}</button>
        </div>
      </div>
      <div v-else class="org-project-settings-empty is-compact">{{ label('emptySorts') }}</div>
    </div>
  </section>

  <section id="org-project-settings-filters" class="ui segment org-project-settings-panel" aria-labelledby="org-project-settings-filters-title">
    <header class="org-project-settings-panel-heading">
      <div>
        <h3 id="org-project-settings-filters-title">{{ label('filters') }}</h3>
        <p>{{ label('filtersDescription') }}</p>
      </div>
      <button type="button" class="ui small button" :disabled="!activeFields.length" @click="addFilter">{{ label('addFilter') }}</button>
    </header>
    <div v-if="filters.length" class="org-project-rule-list org-project-settings-panel-body">
      <div v-for="(filter, index) in filters" :key="index" class="org-project-rule org-project-filter-rule">
        <span class="org-project-rule-index">{{ index + 1 }}</span>
        <div class="field"><label>{{ label('key') }}</label><input v-model.trim="filter.key"></div>
        <div class="field"><label>{{ label('label') }}</label><input v-model.trim="filter.label"></div>
        <div class="field"><label>{{ label('field') }}</label><select v-model="filter.field_key"><option v-for="field in activeFields" :key="field.key" :value="field.key">{{ field.label }}</option></select></div>
        <div class="field"><label>{{ label('operator') }}</label><select v-model="filter.operator"><option value="eq">{{ label('equals') }}</option><option value="ne">{{ label('notEqual') }}</option><option value="contains">{{ label('contains') }}</option><option value="is_empty">{{ label('isEmpty') }}</option><option value="is_not_empty">{{ label('isNotEmpty') }}</option><option value="gte">{{ label('atLeast') }}</option><option value="lte">{{ label('atMost') }}</option><option value="member">{{ label('containsMember') }}</option></select></div>
        <button type="button" class="ui tiny basic red button org-project-rule-remove" :aria-label="label('removeFilter')" @click="filters.splice(index, 1)">{{ label('remove') }}</button>
      </div>
    </div>
    <div v-else class="org-project-settings-empty">{{ label('emptyFilters') }}</div>
  </section>
</template>
