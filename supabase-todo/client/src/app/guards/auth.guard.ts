import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { SupabaseService } from '../supabase.service';

export const authGuard: CanActivateFn = async () => {
  const supabase = inject(SupabaseService);
  const router = inject(Router);
  const user = await supabase.getUser();
  if (user) return true;
  return router.createUrlTree(['/sign-in']);
};

export const guestGuard: CanActivateFn = async () => {
  const supabase = inject(SupabaseService);
  const router = inject(Router);
  const user = await supabase.getUser();
  if (!user) return true;
  return router.createUrlTree(['/todos']);
};
