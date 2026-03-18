<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

   <?spec xslt#document?>

<xsl:template match="/">
  <out xml:base="./xgespr%C3%A4ch.xml">
      <!-- test use of an escaped base URI with document() function -->
      <xsl:copy-of select="document('')/*"/>
  </out>
</xsl:template>

</xsl:stylesheet>
