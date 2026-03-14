<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema" exclude-result-prefixes="xs">

  <!-- Ensure with-param elements don't have access to new values of parameters -->


  <xsl:template match="/">
    <xsl:apply-templates select="//page">
      <xsl:with-param name="max" select="1000"/>
    </xsl:apply-templates>
  </xsl:template>

  <xsl:template match="page">
    <xsl:param name="max" as="xs:integer"/>
    <out>
      <xsl:iterate select="*">
        <xsl:param name="p" select="0"/>
        <xsl:param name="q" select="0"/>
        <xsl:choose>
          <xsl:when test="self::h3">
            <xsl:break>
              <exit at="{position()}" of="{last()}"/>
            </xsl:break>
          </xsl:when>
          <xsl:otherwise>
            <z p="{$p}" q="{$q}"/>
            <xsl:next-iteration>
              <xsl:with-param name="p" select="$p + 1"/>
              <xsl:with-param name="q" select="$p + 1"/>
              <!-- this is the old value of $p -->
            </xsl:next-iteration>
          </xsl:otherwise>
        </xsl:choose>
      </xsl:iterate>
    </out>
  </xsl:template>

  <xsl:template match="p">
    <xsl:copy-of select="."/>
  </xsl:template>

</xsl:stylesheet>
