<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-arithmetic?>
    <!-- Purpose: Test of unary '-' on a union. -->

<xsl:template match="doc">
  <out>
    <xsl:copy-of select="-((n1|n2)[1])"/>
  </out>
</xsl:template>

</xsl:stylesheet>
