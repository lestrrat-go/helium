<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-general-comparisons?>
    <!-- Purpose: Test of "<" operator comparing real to integer, true result -->

<xsl:template match="doc">
  <out>
    <xsl:value-of select="1.9999999 &lt; 2"/>
  </out>
</xsl:template>

</xsl:stylesheet>