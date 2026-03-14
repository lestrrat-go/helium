<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-numeric-subtract?>
    <!-- Purpose: Test of sum() with unary - in some nodes. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="sum(e)"/>
  </out>
</xsl:template>

</xsl:stylesheet>
