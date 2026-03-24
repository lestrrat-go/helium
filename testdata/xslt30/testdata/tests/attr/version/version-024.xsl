<xsl:stylesheet version="25.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<!-- Unrecognized top-level elements are ignored in forwards-compatibility mode
      *along with their content* -->
      <?spec xslt#forwards?>


<xsl:template match="/">
  <boo/>
</xsl:template>

<xsl:subroutine match="*">
  <xsl:copy-of/>
  <xsl:jamaica/>
  <xsl:variable name="A nice cup of tea" version="2.0"/>
  <xsl:choose>
    <xsl:always>Put the kettle on</xsl:always>
  </xsl:choose>
</xsl:subroutine>

</xsl:stylesheet>