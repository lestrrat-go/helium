<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-boolean?>
    <!-- Purpose: Test of boolean() on string "1". -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="doc">
  <out>
    <xsl:value-of select="boolean('1')"/>
  </out>
</xsl:template>

</xsl:stylesheet>