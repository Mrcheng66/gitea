import {attachSearchBox} from './search.ts';
import {GET} from './fetch.ts';

vi.mock('./fetch.ts', () => ({
  GET: vi.fn(),
}));

describe('search box', {concurrent: false}, () => {
  beforeEach(() => {
    document.body.innerHTML = '<div class="ui search"><input class="prompt"><div class="results"></div></div>';
    vi.mocked(GET).mockReset();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  test('returns the selected result to callers while writing its title to the input', async () => {
    vi.mocked(GET).mockResolvedValue({ok: true, json: async () => ({ok: true})} as Response);
    const container = document.querySelector<HTMLElement>('.ui.search')!;
    const input = container.querySelector<HTMLInputElement>('input')!;
    const onSelect = vi.fn();

    attachSearchBox(container, '/search?q={query}', () => [{title: 'org/repo', value: '42'}], {onSelect});
    input.value = 'repo';
    input.dispatchEvent(new Event('input', {bubbles: true}));
    await vi.advanceTimersByTimeAsync(200);
    container.querySelector<HTMLElement>('.result')!.dispatchEvent(new MouseEvent('mousedown', {bubbles: true}));

    expect(input.value).toBe('org/repo');
    expect(onSelect).toHaveBeenCalledWith({title: 'org/repo', value: '42'});
  });

  test('can search after an outside click dismisses an earlier request', async () => {
    document.body.insertAdjacentHTML('beforeend', '<button type="button">Outside</button>');
    vi.mocked(GET).mockResolvedValue({ok: true, json: async () => ({ok: true})} as Response);
    const container = document.querySelector<HTMLElement>('.ui.search')!;
    const input = container.querySelector<HTMLInputElement>('input')!;

    attachSearchBox(container, '/search?q={query}', () => [{title: 'repo'}]);
    input.value = 'first';
    input.dispatchEvent(new Event('input', {bubbles: true}));
    document.querySelector<HTMLButtonElement>('button')!.click();
    await vi.advanceTimersByTimeAsync(200);
    expect(GET).not.toHaveBeenCalled();

    input.value = 'repo';
    input.dispatchEvent(new Event('input', {bubbles: true}));
    await vi.advanceTimersByTimeAsync(200);
    expect(GET).toHaveBeenCalledWith('/search?q=repo', expect.any(Object));
  });

  test('keeps the existing title-only selection behavior without a callback', async () => {
    vi.mocked(GET).mockResolvedValue({ok: true, json: async () => ({ok: true})} as Response);
    const container = document.querySelector<HTMLElement>('.ui.search')!;
    const input = container.querySelector<HTMLInputElement>('input')!;

    attachSearchBox(container, '/search?q={query}', () => [{title: 'repo'}]);
    input.value = 'repo';
    input.dispatchEvent(new Event('input', {bubbles: true}));
    await vi.advanceTimersByTimeAsync(200);
    container.querySelector<HTMLElement>('.result')!.dispatchEvent(new MouseEvent('mousedown', {bubbles: true}));

    expect(input.value).toBe('repo');
  });
});
