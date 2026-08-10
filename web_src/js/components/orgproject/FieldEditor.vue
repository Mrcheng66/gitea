<script setup lang="ts">
import type {OrgProjectField, OrgProjectFieldType} from './types.ts';

const props = defineProps<{labels: Record<string, string>}>();
const fields = defineModel<OrgProjectField[]>({required: true});

const fieldTypes: OrgProjectFieldType[] = [
  'short_text', 'long_text', 'single_select', 'multi_select', 'integer', 'decimal',
  'percent', 'date', 'date_time', 'boolean', 'member', 'member_array',
];

function label(key: string) {
  return props.labels[key] || key;
}

function format(key: string, value: number) {
  return label(key).replace('{n}', String(value));
}

function addField() {
  const index = fields.value.length + 1;
  fields.value.push({key: `field_${index}`, label: format('newField', index), type: 'short_text', order: fields.value.length});
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
  field.options.push({key: `option_${index}`, label: format('newOption', index), order: field.options.length});
}
</script>

<template>
  <section id="org-project-settings-fields" class="ui segment org-project-settings-panel" aria-labelledby="org-project-settings-fields-title">
    <header class="org-project-settings-panel-heading">
      <div>
        <h3 id="org-project-settings-fields-title">{{ label('fields') }}</h3>
        <p>{{ label('fieldsDescription') }}</p>
      </div>
      <div class="org-project-settings-panel-tools">
        <span class="org-project-settings-count">{{ format('fieldCount', fields.length) }}</span>
        <button type="button" class="ui small primary button" @click="addField">{{ label('addField') }}</button>
      </div>
    </header>
    <div v-if="fields.length" class="org-project-editor-list">
      <details v-for="(field, index) in fields" :key="`${field.key}-${index}`" class="org-project-field-item" :class="{disabled: field.archived}" :open="index === 0">
        <summary class="org-project-field-summary">
          <span class="org-project-rule-index">{{ String(index + 1).padStart(2, '0') }}</span>
          <span class="org-project-field-identity">
            <strong>{{ field.label || label('untitledField') }}</strong>
            <small>{{ field.key || label('emptyKey') }}</small>
          </span>
          <span class="org-project-field-type">{{ label(`type_${field.type}`) }}</span>
          <span v-if="field.required" class="org-project-field-state">{{ label('required') }}</span>
          <span v-if="field.archived" class="org-project-field-state is-archived">{{ label('archived') }}</span>
        </summary>
        <div class="org-project-field-body">
          <div class="org-project-editor-grid org-project-field-grid">
            <div class="field"><label>{{ label('key') }}</label><input v-model.trim="field.key" :disabled="field.archived"></div>
            <div class="field"><label>{{ label('label') }}</label><input v-model.trim="field.label" :disabled="field.archived"></div>
            <div class="field"><label>{{ label('type') }}</label><select v-model="field.type" class="ui fluid selection dropdown org-project-dropdown" :disabled="field.archived"><option v-for="type in fieldTypes" :key="type" :value="type">{{ label(`type_${type}`) }}</option></select></div>
            <label class="org-project-inline-check"><input v-model="field.required" type="checkbox" :disabled="field.archived"> {{ label('required') }}</label>
          </div>
          <div v-if="field.type === 'single_select' || field.type === 'multi_select'" class="org-project-field-options">
            <div class="org-project-field-subheading">
              <strong>{{ label('options') }}</strong>
              <button type="button" class="ui tiny button" :disabled="field.archived" @click="addOption(field)">{{ label('addOption') }}</button>
            </div>
            <div v-for="(option, optionIndex) in field.options" :key="optionIndex" class="org-project-option-row">
              <span class="org-project-rule-index">{{ optionIndex + 1 }}</span>
              <div class="field"><label>{{ label('optionKey') }}</label><input v-model.trim="option.key" :disabled="field.archived"></div>
              <div class="field"><label>{{ label('optionLabel') }}</label><input v-model.trim="option.label" :disabled="field.archived"></div>
              <button type="button" class="ui tiny basic red button org-project-rule-remove" :aria-label="label('removeOption')" :disabled="field.archived" @click="field.options!.splice(optionIndex, 1)">{{ label('remove') }}</button>
            </div>
          </div>
          <footer class="org-project-field-actions">
            <div class="org-project-order-actions">
              <button type="button" class="ui tiny basic button" :disabled="index === 0" @click="move(index, -1)">{{ label('moveUp') }}</button>
              <button type="button" class="ui tiny basic button" :disabled="index === fields.length - 1" @click="move(index, 1)">{{ label('moveDown') }}</button>
            </div>
            <button v-if="field.archived" type="button" class="ui tiny button" @click="restore(index)">{{ label('restore') }}</button>
            <button v-else type="button" class="ui tiny basic red button" @click="archiveOrRemove(index)">{{ label('archive') }}</button>
          </footer>
        </div>
      </details>
    </div>
    <div v-else class="org-project-settings-empty">{{ label('emptyFields') }}</div>
  </section>
</template>
