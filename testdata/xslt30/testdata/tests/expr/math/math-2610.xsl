<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec fo#func-round?>
<?spec fo#func-floor?>
<?spec fo#func-ceiling?>
<!-- PURPOSE:  Test round(), floor(), ceiling() applied to a decimal -->

<xsl:param name="x" select="12345678901234.12345"/>
<xsl:param name="y" select="17.5"/>
<xsl:param name="z" select="-17.5"/>

<xsl:template match="doc">
<r>
<out test="x">
<a val="{round($x)}" check="{round($x) instance of xs:decimal}"/>;
<a val="{floor($x)}" check="{floor($x) instance of xs:decimal}"/>;
<a val="{ceiling($x)}" check="{ceiling($x) instance of xs:decimal}"/>;
</out>
<out test="y">
<a val="{round($y)}" check="{round($y) instance of xs:decimal}"/>;
<a val="{floor($y)}" check="{floor($y) instance of xs:decimal}"/>;
<a val="{ceiling($y)}" check="{ceiling($y) instance of xs:decimal}"/>;
</out>
<out test="z">
<a val="{round($z)}" check="{round($z) instance of xs:decimal}"/>;
<a val="{floor($z)}" check="{floor($z) instance of xs:decimal}"/>;
<a val="{ceiling($z)}" check="{ceiling($z) instance of xs:decimal}"/>;
</out>
</r>
</xsl:template>
</xsl:stylesheet>