import { apiClient } from "./client";
import type { ListKindleBooksResponse } from "../types/kindle";

export async function listKindleBooks(): Promise<ListKindleBooksResponse> {
  const res = await apiClient.get<ListKindleBooksResponse>("/highlights/books");
  return res.data;
}
