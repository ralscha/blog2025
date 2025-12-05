import { Component } from '@angular/core';
import { CurrencyPipe, DatePipe, PercentPipe } from '@angular/common';

@Component({
  selector: 'app-home',
  imports: [CurrencyPipe, DatePipe, PercentPipe],
  templateUrl: './home.html'
})
export class Home {
  name = 'Alex';
  gender: 'male' | 'female' | 'other' = 'other';
  unreadCount = 1;
  itemCount = 2;
  today = new Date();
  price = 1234.56;
  percent = 0.42;
  items = [
    $localize`:fruitApples:Apples`,
    $localize`:fruitBananas:Bananas`,
    $localize`:fruitCherries:Cherries`
  ];
  greeting = $localize`Hello ${this.name}, you have ${this.itemCount} items in your cart.`;
}
