<?xml version="1.0"?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Test the static-base-uri() function with no parameters -->


  <xsl:template match="doc" xml:base="http://some.other/">
    <out>
      <xsl:value-of select="static-base-uri()"/>
    </out>
  </xsl:template>

</xsl:stylesheet>
