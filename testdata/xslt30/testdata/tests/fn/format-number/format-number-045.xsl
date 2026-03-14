<?xml version="1.0"?>
<?spec xslt#format-number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- format-number() against an empty sequence -->



<xsl:template match="doc">
<out>
<xsl:value-of select="format-number((),'###.###')"/>
</out>
</xsl:template>

</xsl:stylesheet>
