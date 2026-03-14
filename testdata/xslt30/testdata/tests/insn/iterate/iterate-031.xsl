<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <!-- Basic test of xsl:iterate; exercises xsl:on-completion and implicit continuation -->

  <xsl:template match="/">
    <xsl:apply-templates select="//page">
      <xsl:with-param name="max" select="1000"/>
    </xsl:apply-templates>
  </xsl:template>

  <xsl:template match="page">
    <xsl:param name="max" as="xs:integer"/>
    <out>
      <xsl:iterate select="*">
        <xsl:param name="count" select="0"/>
        <xsl:on-completion select="'count ' || $count">
          <quit/>
        </xsl:on-completion>
        <xsl:fallback/>
        <xsl:choose>
          <xsl:when test="self::h3">
            <heading>
              <xsl:copy-of select="child::node()"/>
            </heading>
            <xsl:next-iteration>
              <xsl:with-param name="count" select="$count+1"/>
            </xsl:next-iteration>
          </xsl:when>
          <xsl:otherwise>
            <xsl:copy-of select="."/>
          </xsl:otherwise>
        </xsl:choose>     
        <xsl:fallback/>
      </xsl:iterate>
    </out>
  </xsl:template>

  <xsl:template match="p">
    <xsl:copy-of select="."/>
  </xsl:template>

</xsl:stylesheet>
