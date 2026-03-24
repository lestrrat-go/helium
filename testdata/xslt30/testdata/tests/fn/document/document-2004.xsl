<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

   <?spec xslt#document?>

<xsl:template match="/">
  <out>
      <!-- test use of an escaped URI with document() function -->
      <xsl:copy-of select="document('xgespr%C3%A4ch.xml')/*"/>
  </out>
</xsl:template>

</xsl:stylesheet>
