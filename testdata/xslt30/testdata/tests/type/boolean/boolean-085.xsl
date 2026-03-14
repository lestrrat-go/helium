<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-general-comparisons?>
    <!-- Purpose: Test of the '!=' operator returning false on two numbers. -->

<xsl:template match="/">
  <out>
    <xsl:value-of select="1!=1"/>
  </out>
</xsl:template>

</xsl:stylesheet>