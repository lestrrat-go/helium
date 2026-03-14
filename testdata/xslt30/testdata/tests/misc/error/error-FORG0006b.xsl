<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

<!-- PURPOSE:  Test how sum() handles a mixed sequence -->
<?spec fo#func-error?><?error FORG0006?>

<xsl:param name="p" select="7"/>
<xsl:param name="int" select="(1, 2, 3, $p, 'fred')"/>


<xsl:template match="doc">
<out>
<a val="{sum($int)}"/>;
</out>
</xsl:template>
</xsl:stylesheet>