<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#id-path-expressions?>
    <!-- Purpose: Test that combination of // and descendant specifies node can be anywhere in ancestry. -->

<xsl:template match="doc">
  <out>

  <xsl:for-each select="//xx/descendant::*">
    <xsl:value-of select="name(.)"/><xsl:text> </xsl:text>
  </xsl:for-each>

  </out>
</xsl:template>

</xsl:stylesheet>