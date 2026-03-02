import { Injectable } from '@angular/core';
import {
  AuthChangeEvent,
  createClient,
  Session,
  SupabaseClient,
  User,
} from '@supabase/supabase-js';
import { environment } from '../environments/environment';

export interface Profile {
  avatar_url: string;
}

export interface Todo {
  id?: number;
  user_id?: string;
  title: string;
  description: string | null;
  is_complete: boolean;
  priority: 'low' | 'medium' | 'high';
  due_date: string | null;
  inserted_at?: string;
  updated_at?: string;
}

type TodoWriteColumn = 'title' | 'description' | 'is_complete' | 'priority' | 'due_date';

type TodoInsertPayload = Pick<Todo, TodoWriteColumn>;
type TodoUpdatePayload = Partial<Pick<Todo, TodoWriteColumn>>;

const TODO_READ_COLUMNS_SQL = 'id,title,description,is_complete,priority,due_date';

@Injectable({
  providedIn: 'root',
})
export class SupabaseService {
  private supabase: SupabaseClient;

  constructor() {
    this.supabase = createClient(environment.supabaseUrl, environment.supabaseKey);
  }

  // Auth

  async getUser(): Promise<User | null> {
    const { data, error } = await this.supabase.auth.getUser();
    if (error) return null;
    return data.user;
  }

  signUp(email: string, password: string) {
    return this.supabase.auth.signUp({ email, password });
  }

  signIn(email: string, password: string) {
    return this.supabase.auth.signInWithPassword({ email, password });
  }

  signOut() {
    return this.supabase.auth.signOut();
  }

  authChanges(callback: (event: AuthChangeEvent, session: Session | null) => void) {
    return this.supabase.auth.onAuthStateChange(callback);
  }

  // Profile

  profile() {
    return this.supabase.from('profiles').select('avatar_url').single();
  }

  updateProfile(profile: Profile, userId: string) {
    return this.supabase.from('profiles').update(profile).eq('id', userId);
  }

  // Avatar (Storage)

  downloadImage(path: string) {
    return this.supabase.storage.from('avatars').download(path);
  }

  uploadAvatar(filePath: string, file: File) {
    return this.supabase.storage.from('avatars').upload(filePath, file);
  }

  removeAvatar(filePath: string) {
    return this.supabase.storage.from('avatars').remove([filePath]);
  }

  // Todos

  getTodos() {
    return this.supabase
      .from('todos')
      .select(TODO_READ_COLUMNS_SQL)
      .order('inserted_at', { ascending: false });
  }

  getTodo(id: number) {
    return this.supabase.from('todos').select(TODO_READ_COLUMNS_SQL).eq('id', id).single();
  }

  addTodo(todo: TodoInsertPayload) {
    return this.supabase.from('todos').insert(todo);
  }

  updateTodo(id: number, changes: TodoUpdatePayload) {
    return this.supabase.from('todos').update(changes).eq('id', id);
  }

  deleteTodo(id: number) {
    return this.supabase.from('todos').delete().eq('id', id);
  }
}
