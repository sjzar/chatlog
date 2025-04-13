import { useState } from "react";

export type RequestService<TRequestParameters, TResponse> = (params?: TRequestParameters) => Promise<TResponse>;
export type UseRequestResult<TRequestParameters, TResponse> = {
  /**
   * 当前请求状态
   */
  loading: boolean;

  /**
   * 执行 http 请求
   */
  run: RequestService<TRequestParameters, TResponse>;
};

/**
 * http request hook.
 */
export function useRequest<TRequestParameters, TResponse>(callback: RequestService<TRequestParameters, TResponse>): UseRequestResult<TRequestParameters, TResponse> {
  const [loading, setLoading] = useState(false);

  async function run(params?: TRequestParameters): Promise<TResponse> {
    setLoading(true);

    try {
      const data = await callback(params);
      setLoading(false);
      return data;
    } catch (err) {
      setLoading(false);
      throw err;
    }
  }

  return  {
    loading,
    run
  };
}
