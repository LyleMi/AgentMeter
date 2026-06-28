import { ref, shallowRef, type Ref, type ShallowRef } from 'vue'

export interface AsyncResource<T> {
  data: ShallowRef<T>
  loading: Ref<boolean>
  error: Ref<string>
  run: (loader: () => Promise<T>, options?: AsyncRunOptions<T>) => Promise<T | undefined>
  clearError: () => void
}

export interface AsyncRunOptions<T> {
  onErrorData?: T | (() => T)
}

export function useAsyncResource<T>(initialData: T): AsyncResource<T> {
  const data = shallowRef(initialData) as ShallowRef<T>
  const loading = ref(false)
  const error = ref('')
  let requestId = 0

  async function run(loader: () => Promise<T>, options: AsyncRunOptions<T> = {}) {
    const currentRequest = ++requestId
    loading.value = true
    error.value = ''
    try {
      const next = await loader()
      if (currentRequest === requestId) data.value = next
      return next
    } catch (loadError) {
      if (currentRequest === requestId) {
        error.value = loadError instanceof Error ? loadError.message : String(loadError)
        if (options.onErrorData !== undefined) {
          data.value = typeof options.onErrorData === 'function' ? (options.onErrorData as () => T)() : options.onErrorData
        }
      }
      return undefined
    } finally {
      if (currentRequest === requestId) loading.value = false
    }
  }

  return {
    data,
    loading,
    error,
    run,
    clearError: () => {
      error.value = ''
    }
  }
}
