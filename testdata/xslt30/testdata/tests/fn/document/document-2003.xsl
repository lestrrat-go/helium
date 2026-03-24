<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

   <?spec xslt#document?>

<xsl:template match="/">
  <out xml:base="..">
      <!-- test use of xml:base with document() function -->
      <xsl:copy-of select="document('document/doc14.xml')/*"/>
  </out>
</xsl:template>

</xsl:stylesheet>
