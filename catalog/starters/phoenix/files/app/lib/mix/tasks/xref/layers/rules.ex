defmodule Mix.Tasks.Xref.Layers.Rules do
  def violations(sources, layers, opts \\ []) do
    ignored = MapSet.new(Keyword.get(opts, :ignore, []))
    exported = MapSet.new(Keyword.get(opts, :exports, []))
    may_use = Map.new(Keyword.get(opts, :may_use, []))
    layer_of = for {layer, modules} <- layers, module <- modules, into: %{}, do: {module, layer}
    order = Enum.flat_map(layers, fn {_layer, modules} -> modules end)
    rank = order |> Enum.with_index() |> Map.new()
    roots = order |> Enum.map(&root/1) |> MapSet.new()
    files = Enum.map(sources, fn {path, code} -> parse(path, code, roots) end)
    schemas = files |> Enum.flat_map(& &1.schemas) |> MapSet.new()
    public = MapSet.union(schemas, exported)

    context = %{
      rank: rank,
      roots: roots,
      public: public,
      exported: exported,
      ignored: ignored,
      may_use: may_use,
      layer_of: layer_of
    }

    files
    |> Enum.flat_map(&check_file(&1, context))
    |> Enum.uniq()
  end

  def describe(%{kind: :points_up} = v),
    do:
      "#{v.file}:#{v.line}: #{inspect(v.from)} depends on #{inspect(v.to)}, which is higher in the layer list"

  def describe(%{kind: :past_top_module} = v),
    do:
      "#{v.file}:#{v.line}: #{inspect(v.from)} uses #{inspect(v.to)} inside another module. Call its top module, or use only its schemas and exports"

  def describe(%{kind: :not_allowed} = v),
    do:
      "#{v.file}:#{v.line}: #{inspect(v.from)} uses #{inspect(v.to)}. The #{v.layer} layer may use only exported ports and the layers #{Enum.join(v.allowed, ", ")}"

  def describe(%{kind: :not_in_list} = v),
    do: "#{v.file}:#{v.line}: #{inspect(v.to)} has no place in the layer list"

  defp check_file(file, context) do
    Enum.flat_map(file.modules, fn {module, line, references} ->
      if MapSet.member?(context.ignored, module) do
        []
      else
        check_module(file.path, module, line, references, context)
      end
    end)
  end

  defp check_module(path, module, line, references, context) do
    case owner(module, context.rank) do
      nil ->
        [%{file: path, line: line, kind: :not_in_list, from: module, to: module}]

      source ->
        Enum.flat_map(references, fn {target, ref_line} ->
          check_reference(path, ref_line, source, target, context)
        end)
    end
  end

  defp check_reference(path, line, source, target, context) do
    violation = %{file: path, line: line, from: source, to: target}

    case owner(target, context.rank) do
      nil ->
        if MapSet.member?(context.ignored, target),
          do: [],
          else: [Map.put(violation, :kind, :not_in_list)]

      ^source ->
        []

      target_owner ->
        up = if context.rank[target_owner] < context.rank[source], do: [:points_up], else: []

        inside =
          if target != target_owner and not MapSet.member?(context.public, target),
            do: [:past_top_module],
            else: []

        Enum.map(up ++ inside, &Map.put(violation, :kind, &1)) ++
          limit(violation, source, target, target_owner, context)
    end
  end

  defp limit(violation, source, target, target_owner, context) do
    layer = context.layer_of[source]

    case Map.fetch(context.may_use, layer) do
      :error ->
        []

      {:ok, allowed} ->
        if MapSet.member?(context.exported, target) or context.layer_of[target_owner] in allowed,
          do: [],
          else: [Map.merge(violation, %{kind: :not_allowed, layer: layer, allowed: allowed})]
    end
  end

  defp owner(module, rank) do
    module
    |> Module.split()
    |> prefixes()
    |> Enum.find(&Map.has_key?(rank, &1))
  end

  defp prefixes(segments) do
    for n <- length(segments)..1//-1, do: segments |> Enum.take(n) |> Module.concat()
  end

  defp root(module), do: module |> Module.split() |> hd()

  defp parse(path, code, roots) do
    {:ok, ast} = Code.string_to_quoted(code, file: path)

    modules =
      for {:defmodule, meta, [{:__aliases__, _, segments}, [do: body]]} <- top_level(ast) do
        module = Module.concat(segments)
        {module, meta[:line], references(body, module, roots)}
      end

    schemas =
      for {:defmodule, _, [{:__aliases__, _, segments}, [do: body]]} <- top_level(ast),
          uses_ecto_schema?(body),
          do: Module.concat(segments)

    %{path: path, modules: modules, schemas: schemas}
  end

  defp top_level({:__block__, _, forms}), do: forms
  defp top_level(form), do: [form]

  defp uses_ecto_schema?(body) do
    {_body, found?} =
      Macro.prewalk(body, false, fn
        {:use, _, [{:__aliases__, _, [:Ecto, :Schema]} | _]} = node, _found -> {node, true}
        node, found -> {node, found}
      end)

    found?
  end

  defp references(body, module, roots) do
    aliases = aliases(body, module)

    {_body, found} =
      Macro.prewalk(body, [], fn
        {:alias, meta, [{{:., _, [{:__aliases__, _, base}, :{}]}, _, children}]}, acc ->
          names =
            for {:__aliases__, _, segments} <- children,
                do: {Module.concat(base ++ segments), meta[:line]}

          {:ok, names ++ acc}

        {:__aliases__, meta, segments} = node, acc ->
          case expand(segments, aliases, module) do
            nil -> {node, acc}
            name -> {node, [{name, meta[:line]} | acc]}
          end

        node, acc ->
          {node, acc}
      end)

    found
    |> Enum.filter(fn {name, _line} -> MapSet.member?(roots, root(name)) end)
    |> Enum.reverse()
  end

  defp aliases(body, module) do
    {_body, found} =
      Macro.prewalk(body, %{}, fn
        {:alias, _, [{{:., _, [{:__aliases__, _, base}, :{}]}, _, children}]} = node, acc ->
          added =
            for {:__aliases__, _, segments} <- children,
                into: %{},
                do: {List.last(segments), base ++ segments}

          {node, Map.merge(acc, added)}

        {:alias, _, [{:__aliases__, _, segments} | opts]} = node, acc ->
          full = expand_segments(segments, acc, module)
          {node, Map.put(acc, alias_key(full, opts), full)}

        node, acc ->
          {node, acc}
      end)

    found
  end

  defp alias_key(_full, [[as: {:__aliases__, _, [as]}]]), do: as
  defp alias_key(full, _opts), do: List.last(full)

  defp expand([{:__MODULE__, _, _} | rest], _aliases, module), do: Module.concat([module | rest])

  defp expand(segments, aliases, module) do
    if Enum.all?(segments, &is_atom/1),
      do: Module.concat(expand_segments(segments, aliases, module))
  end

  defp expand_segments([first | rest] = segments, aliases, _module) do
    case Map.fetch(aliases, first) do
      {:ok, full} -> full ++ rest
      :error -> segments
    end
  end
end
