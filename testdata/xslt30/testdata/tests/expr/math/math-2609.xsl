<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<?spec fo#func-round?>
<?spec fo#func-floor?>
<?spec fo#func-ceiling?>
<!-- PURPOSE:  Test round(), floor(), ceiling() applied to an integer -->

<xsl:param name="x" select="12345678901234"/>

<xsl:template match="doc">
<out>
<a val="{round($x)}" check="{round($x) instance of xs:integer}"/>;
<a val="{floor($x)}" check="{floor($x) instance of xs:integer}"/>;
<a val="{ceiling($x)}" check="{ceiling($x) instance of xs:integer}"/>;
</out>
</xsl:template>
</xsl:stylesheet>