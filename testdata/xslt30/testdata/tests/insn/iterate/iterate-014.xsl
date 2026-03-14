<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <!-- Basic test of xsl:iterate; uses variables -->


  <xsl:template match="/">
    <xsl:apply-templates select="//page">
      <xsl:with-param name="max" select="1000"/>
    </xsl:apply-templates>
  </xsl:template>

  <xsl:template match="page">
    <xsl:param name="max" as="xs:integer"/>
    <out>
      <xsl:iterate select="p">
        <xsl:param name="space-used" select="0"/>
        <xsl:copy-of select="."/>
        <xsl:variable name="new-space-used" select="$space-used + string-length(.)"/>
        <xsl:if test="$new-space-used gt $max">
          <page-break/>
        </xsl:if>
        <xsl:next-iteration>
          <xsl:with-param name="space-used"
            select="if ($new-space-used gt $max) then 0 else $new-space-used"/>
        </xsl:next-iteration>
      </xsl:iterate>
    </out>
  </xsl:template>

</xsl:stylesheet>
