import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { Router } from '@angular/router';
import { LocaleDatePipe } from '../locale-date.pipe';
import { SupabaseService, Todo } from '../supabase.service';

@Component({
  selector: 'app-todo-list',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [LocaleDatePipe],
  templateUrl: './todo-list.html',
})
export class TodoListComponent implements OnInit {
  private readonly supabase = inject(SupabaseService);
  private readonly router = inject(Router);

  todos = signal<Todo[]>([]);
  loading = signal(true);
  showCompleted = signal(true);

  completedCount = computed(() => this.todos().filter((t) => t.is_complete).length);
  visibleTodos = computed(() =>
    this.showCompleted() ? this.todos() : this.todos().filter((t) => !t.is_complete),
  );

  async ngOnInit() {
    await this.loadTodos();
  }

  private async loadTodos() {
    this.loading.set(true);
    const { data, error } = await this.supabase.getTodos();
    this.loading.set(false);

    if (error) {
      console.error('Error loading todos:', error.message);
      return;
    }

    this.todos.set(data ?? []);
  }

  async toggleComplete(todo: Todo) {
    const updated = !todo.is_complete;
    this.todos.update((list) =>
      list.map((t) => (t.id === todo.id ? { ...t, is_complete: updated } : t)),
    );
    const { error } = await this.supabase.updateTodo(todo.id!, { is_complete: updated });
    if (error) {
      this.todos.update((list) =>
        list.map((t) => (t.id === todo.id ? { ...t, is_complete: !updated } : t)),
      );
    }
  }

  goToNew() {
    this.router.navigate(['/todos', 'new']);
  }

  goToEdit(todo: Todo) {
    this.router.navigate(['/todos', todo.id]);
  }
}
