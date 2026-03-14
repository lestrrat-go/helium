<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec xslt#generate-id?>
<!-- PURPOSE:   Test of 'generate-id()' - make sure same node generates same ID. -->
<xsl:template match="doc">
<out>
<xsl:value-of select="generate-id(a)=generate-id(a) and not(generate-id(a1)=generate-id(b)) and not(generate-id(c))"/>
</out>
</xsl:template>
</xsl:stylesheet>