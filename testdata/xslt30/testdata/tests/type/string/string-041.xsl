<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec fo#func-string?>
<!-- PURPOSE:   Test of 'string()' conversion function. -->
<xsl:variable name="ResultTreeFragTest">
   <xsl:value-of select="doc"/>
</xsl:variable>

<xsl:template match="doc">
<out>
<xsl:value-of select="string($ResultTreeFragTest)"/>
</out>
</xsl:template>
</xsl:stylesheet>