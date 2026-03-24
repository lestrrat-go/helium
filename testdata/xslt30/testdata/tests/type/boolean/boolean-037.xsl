<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">
<?spec fo#func-not?>
<!--  Test of boolean "not" operator -->
<xsl:template match="doc">
<out>
<xsl:value-of select="not('00')"/>
</out>
</xsl:template>
</xsl:stylesheet>