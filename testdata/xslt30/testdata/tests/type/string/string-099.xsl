<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec fo#func-translate?>
<!-- PURPOSE:   Test of 'translate()' function. -->
<xsl:template match="doc">
<out>
<xsl:value-of select="translate(&#34;bar&#34;,&#34;RAB&#34;,&#34;xyz&#34;)"/>
</out>
</xsl:template>
</xsl:stylesheet>