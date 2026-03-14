<?spec xpath#id-predicates?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  version="2.0">

  <!-- FileName: predicate58 -->
  <!-- Document: http://www.w3.org/TR/xpath -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 2.4 -->
    <!-- Purpose: Compare non-existent attributes to ones containing the null string. -->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="doc">
  <out>
      <a desc="has ex: " value="{count(a[@ex])}"/>
      <b desc="has ex, eq null: " value="{count(a[@ex=''])}"/>
      <c desc="has ex, measure null: " value="{count(a[string-length(@ex)=0])}"/>
      <d desc="has ex, neq null: " value="{count(a[@ex!=''])}"/>
      <e desc="has ex, measure non-null: " value="{count(a[string-length(@ex) &gt; 0])}"/>
      <f desc="not has ex: " value="{count(a[not(@ex)])}"/>
      <g desc="not has ex, eq null: " value="{count(a[not(@ex='')])}"/>
      <h desc="not has ex, measure null: " value="{count(a[not(string-length(@ex)=0)])}"/>
      <i desc="has why, eq value: " value="{count(a[@why='value'])}"/>
      <j desc="has why, neq value: " value="{count(a[@why!='value'])}"/>
  </out>
</xsl:template>

</xsl:stylesheet>
