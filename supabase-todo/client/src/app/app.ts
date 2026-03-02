import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
} from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { filter, map } from 'rxjs';
import { SupabaseService } from './supabase.service';

@Component({
  selector: 'app-root',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [RouterOutlet, RouterLink, RouterLinkActive],
  templateUrl: './app.html',
})
export class App implements OnInit {
  private readonly supabase = inject(SupabaseService);
  private readonly router = inject(Router);

  isAuthenticated = signal(false);

  private readonly routerUrl = toSignal(
    this.router.events.pipe(
      filter((e) => e instanceof NavigationEnd),
      map((e) => (e as NavigationEnd).urlAfterRedirects),
    ),
    { initialValue: this.router.url },
  );

  showHamburger = computed(() => !/^\/todos\/.+/.test(this.routerUrl() ?? ''));

  async ngOnInit() {
    const user = await this.supabase.getUser();
    this.isAuthenticated.set(!!user);

    this.supabase.authChanges(async (_event, session) => {
      this.isAuthenticated.set(!!session?.user);
    });
  }

  closeDrawer() {
    const drawer = document.getElementById('app-drawer') as HTMLInputElement | null;
    if (drawer) drawer.checked = false;
  }

  async signOut() {
    this.closeDrawer();
    await this.supabase.signOut();
    this.isAuthenticated.set(false);
    this.router.navigate(['/sign-in']);
  }
}
