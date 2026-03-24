<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-sum?>
    <!-- Purpose: Test of sum() with non-number. -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="sum(for $e in e return number($e))"/>
  </out>
</xsl:template>

</xsl:stylesheet>
