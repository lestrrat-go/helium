<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Test for xsl:key being imported. -->

  <xsl:import href="key-011a.xsl"/>

  <xsl:template match="doc">
    <root>
      <xsl:value-of select="key('mykey', 'Introduction')/p"/>
      <xsl:text> </xsl:text>
      <xsl:value-of select="key('mykey', 'Stylesheet Structure')/p"/>
      <xsl:text> </xsl:text>
      <xsl:value-of select="key('mykey', 'Expressions')/p"/>
    </root>
  </xsl:template>

</xsl:stylesheet>
