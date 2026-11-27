defmodule Mix.Tasks.Xref.LayersTest do
  use ExUnit.Case, async: true

  alias Mix.Tasks.Xref.Layers.Rules

  @layers [
    inbound: [Shop.Web],
    outbound: [Shop.Mailer],
    use_cases: [Shop.Checkout],
    contexts: [Shop.Orders, Shop.Catalog],
    base: [Shop.Money]
  ]

  @outbound_limits [exports: [Shop.Checkout.Receipt], may_use: [outbound: [:base]]]

  @order_schema {"lib/shop/orders/order.ex",
                 """
                 defmodule Shop.Orders.Order do
                   use Ecto.Schema
                   schema "orders", do: field(:total, :integer)
                 end
                 """}

  defp violations(sources, opts \\ []) do
    sources
    |> Rules.violations(@layers, opts)
    |> Enum.map(&Map.take(&1, [:kind, :from, :to]))
  end

  describe "violations" do
    test "accepts a dependency down the list, on the top module or on a schema" do
      checkout =
        {"lib/shop/checkout.ex",
         """
         defmodule Shop.Checkout do
           alias Shop.Orders
           alias Shop.Orders.Order

           def run(%Order{} = order), do: Orders.place(order, Shop.Money.zero())
         end
         """}

      assert violations([checkout, @order_schema]) == []
    end

    test "rejects a dependency that points up the list" do
      orders =
        {"lib/shop/orders.ex",
         """
         defmodule Shop.Orders do
           def place(order), do: Shop.Checkout.notify(order)
         end
         """}

      assert violations([orders]) == [%{kind: :points_up, from: Shop.Orders, to: Shop.Checkout}]
    end

    test "rejects a call to a module inside another context, through an alias" do
      checkout =
        {"lib/shop/checkout.ex",
         """
         defmodule Shop.Checkout do
           alias Shop.Orders

           def total(lines), do: Orders.Pricing.total(lines)
         end
         """}

      assert violations([checkout]) == [
               %{kind: :past_top_module, from: Shop.Checkout, to: Shop.Orders.Pricing}
             ]
    end

    test "expands a grouped alias before it checks the module" do
      checkout =
        {"lib/shop/checkout.ex",
         """
         defmodule Shop.Checkout do
           alias Shop.Orders.{Pricing}

           def total(lines), do: Pricing.total(lines)
         end
         """}

      assert Enum.uniq(violations([checkout])) == [
               %{kind: :past_top_module, from: Shop.Checkout, to: Shop.Orders.Pricing}
             ]
    end

    test "accepts an outbound adapter that implements an exported port and uses the base" do
      adapter =
        {"lib/shop/mailer/smtp.ex",
         """
         defmodule Shop.Mailer.Smtp do
           @behaviour Shop.Checkout.Receipt

           def receipt(order), do: Shop.Money.format(order.total)
         end
         """}

      assert violations([adapter], @outbound_limits) == []
    end

    test "rejects an outbound adapter that calls back into the app" do
      adapter =
        {"lib/shop/mailer/smtp.ex",
         """
         defmodule Shop.Mailer.Smtp do
           @behaviour Shop.Checkout.Receipt

           def receipt(order), do: Shop.Orders.customer_email(order)
         end
         """}

      assert violations([adapter], @outbound_limits) == [
               %{kind: :not_allowed, from: Shop.Mailer, to: Shop.Orders}
             ]
    end

    test "lets an inbound adapter call the use cases and the contexts" do
      web =
        {"lib/shop/web.ex",
         """
         defmodule Shop.Web do
           def checkout(params), do: Shop.Checkout.run(params)
           def orders(customer), do: Shop.Orders.list(customer)
         end
         """}

      assert violations([web], @outbound_limits) == []
    end

    test "rejects a use case that names its adapter" do
      checkout =
        {"lib/shop/checkout.ex",
         """
         defmodule Shop.Checkout do
           def send_receipt(order), do: Shop.Mailer.Smtp.receipt(order)
         end
         """}

      assert %{kind: :points_up, from: Shop.Checkout, to: Shop.Mailer.Smtp} in violations([
               checkout
             ])
    end

    test "rejects a module that has no place in the list" do
      shipping =
        {"lib/shop/shipping.ex",
         """
         defmodule Shop.Shipping do
         end
         """}

      assert violations([shipping]) == [
               %{kind: :not_in_list, from: Shop.Shipping, to: Shop.Shipping}
             ]
    end

    test "skips an ignored module, such as the composition root" do
      application =
        {"lib/shop/application.ex",
         """
         defmodule Shop.Application do
           def start(_type, _args), do: Shop.Orders.Supervisor.start_link([])
         end
         """}

      assert violations([application], ignore: [Shop.Application]) == []
    end
  end
end
