<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec fo#func-number?>
<!-- PURPOSE:  Test of number() conversion function. -->
<xsl:variable name="ResultTreeFragTest">
<xsl:value-of select="n4"/>
</xsl:variable>

<xsl:template match="doc">
<out>
<xsl:value-of select="number($ResultTreeFragTest)"/>
</out>
</xsl:template>
</xsl:stylesheet>