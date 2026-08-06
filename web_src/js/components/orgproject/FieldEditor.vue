<script setup lang="ts">
import type {OrgProjectField, OrgProjectFieldType} from './types.ts';

const fields = defineModel<OrgProjectField[]>({required: true});

const fieldTypes: Array<{value: OrgProjectFieldType, label: string}> = [
  {value: 'short_text', label: 'Short text'},
  {value: 'long_text', label: 'Long text'},
  {value: 'single_select', label: 'Single select'},
  {value: 'multi_select', label: 'Multi select'},
  {value: 'integer', label: 'Integer'},
  {value: 'decimal', label: 'Decimal'},
  {value: 'percent', label: 'Percent'},
  {value: 'date', label: 'Date'},
  {value: 'date_time', label: 'Date and time'},
  {value: 'boolean', label: 'Boolean'},
  {value: 'member', label: 'Member'},
  {value: 'member_array', label: 'Members'},
];

function addField() {
  const index = fields.value.length + 1;
  fields.value.push({key: `field_${index}`, label: `Field ${index}`, type: 'short_text', order: fields.value.length});
}

function archiveOrRemove(index: number) {
  const field = fields.value[index]!;
  if (field.archived) fields.value.splice(index, 1);
  else field.archived = true;
}

function restore(index: number) {
  fields.value[index]!.archived = false;
}

function move(index: number, offset: number) {
  const target = index + offset;
  if (target < 0 || target >= fields.value.length) return;
  const [field] = fields.value.splice(index, 1);
  fields.value.splice(target, 0, field!);
}

function addOption(field: OrgProjectField) {
  field.options ??= [];
  const index = field.options.length + 1;
  field.options.push({key: `option_${index}`, label: `Option ${index}`, order: field.options.length});
}
</script>

<template>
  <section class="org-project-editor-section">
    <div class="flex-left-right tw-mb-3"><h3 class="ui header tw-m-0">Fields</h3><button type="button" class="ui small button" @click="addField">Add field</button></div>
    <div class="org-project-editor-list">
      <article v-for="(field, index) in fields" :key="`${field.key}-${index}`" class="ui segment org-project-editor-item" :class="{disabled: field.archived}">
        <div class="org-project-editor-grid">
          <div class="field"><label>Key</label><input v-model.trim="field.key" :disabled="field.archived"></div>
          <div class="field"><label>Label</label><input v-model.trim="field.label" :disabled="field.archived"></div>
          <div class="field"><label>Type</label><select v-model="field.type" :disabled="field.archived"><option v-for="type in fieldTypes" :key="type.value" :value="type.value">{{ type.label }}</option></select></div>
          <label class="org-project-inline-check"><input v-model="field.required" type="checkbox" :disabled="field.archived"> Required</label>
        </div>
        <div v-if="field.type === 'single_select' || field.type === 'multi_select'" class="tw-mt-3">
          <div v-for="(option, optionIndex) in field.options" :key="optionIndex" class="org-project-option-row">
            <input v-model.trim="option.key" aria-label="Option key" :disabled="field.archived">
            <input v-model.trim="option.label" aria-label="Option label" :disabled="field.archived">
            <button type="button" class="ui icon button" aria-label="Remove option" :disabled="field.archived" @click="field.options!.splice(optionIndex, 1)">×</button>
          </div>
          <button type="button" class="ui tiny button tw-mt-2" :disabled="field.archived" @click="addOption(field)">Add option</button>
        </div>
        <div class="flex-text-block tw-mt-3">
          <button type="button" class="ui tiny basic button" :disabled="index === 0" @click="move(index, -1)">Move up</button>
          <button type="button" class="ui tiny basic button" :disabled="index === fields.length - 1" @click="move(index, 1)">Move down</button>
          <button v-if="field.archived" type="button" class="ui tiny button" @click="restore(index)">Restore</button>
          <button v-else type="button" class="ui tiny basic red button" @click="archiveOrRemove(index)">Archive</button>
        </div>
      </article>
    </div>
  </section>
</template>
